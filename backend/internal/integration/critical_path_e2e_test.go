// 关键路径 E2E 集成测试：
//
//	注册新用户 → 创建 API Key → GET /v1/models → POST /v1/chat/completions → usage_logs 落库校验
//
// 整条链路真实执行，不 mock 任何内部逻辑：
//   - testcontainers 启动真实 Postgres + Redis；
//   - 编译并启动真实的 server 二进制（cmd/server），所有步骤走 HTTP；
//   - 唯一的测试替身是"上游 OpenAI"：用本机 httptest 服务假扮上游模型服务
//     （网关内部的鉴权、账号调度、直通转发、计费、usage_logs 写入全部真实运行）。
//
// 运行前提：本机 Docker 可用。运行方式：
//
//	INTEGRATION_TEST=1 go test -v -timeout 600s ./internal/integration/ -run TestCriticalPathE2E
//
// 未设置 INTEGRATION_TEST=1 时自动 skip。
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	_ "github.com/Wei-Shaw/sub2api/ent/runtime"
	dbusagelog "github.com/Wei-Shaw/sub2api/ent/usagelog"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/repository"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

const (
	cpPostgresImage = "postgres:18.1-alpine3.23"
	cpRedisImage    = "redis:8.4-alpine"
	// 与 harness 相同的最低长度要求：config.Load 校验 jwt.secret 非空且 >= 32 字节。
	cpJWTSecret = "critical-path-e2e-jwt-secret-0123456789abcdef"
	cpModel     = "gpt-5.4" // 内置官方定价表中有价，保证 actual_cost > 0
)

// cpEnvelope 是后端统一响应信封 {"code":0,"message":"...","data":...}。
type cpEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type cpAuthData struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	User        struct {
		ID int64 `json:"id"`
	} `json:"user"`
}

type cpAPIKeyData struct {
	ID  int64  `json:"id"`
	Key string `json:"key"`
}

type cpChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

func TestCriticalPathE2E(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("INTEGRATION_TEST=1 未设置，跳过关键路径 E2E 测试")
	}

	ctx := context.Background()
	require.NoError(t, timezone.Init("UTC"))

	// ---------------------------------------------------------------
	// 基础设施：假上游 + Postgres + Redis 容器
	// ---------------------------------------------------------------
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/chat/completions") {
			w.Header().Set("Content-Type", "application/json")
			// usage tokens 必须非零：RecordUsage 对全零 usage 直接丢弃、不写 usage_logs。
			_, _ = io.WriteString(w, `{
				"id": "chatcmpl-critical-path-e2e",
				"object": "chat.completion",
				"model": "`+cpModel+`",
				"choices": [{"index": 0, "message": {"role": "assistant", "content": "pong"}, "finish_reason": "stop"}],
				"usage": {"prompt_tokens": 21, "completion_tokens": 13, "total_tokens": 34}
			}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer upstream.Close()

	pgContainer, err := tcpostgres.Run(
		ctx,
		cpPostgresImage,
		tcpostgres.WithDatabase("sub2api_e2e"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "启动 postgres 容器")
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	redisContainer, err := tcredis.Run(ctx, cpRedisImage)
	require.NoError(t, err, "启动 redis 容器")
	t.Cleanup(func() { _ = redisContainer.Terminate(ctx) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	require.NoError(t, err)
	db := cpOpenSQL(t, ctx, dsn)
	require.NoError(t, repository.ApplyMigrations(ctx, db), "执行数据库迁移")

	entClient := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	t.Cleanup(func() { _ = entClient.Close() })

	// ---------------------------------------------------------------
	// 种子数据（业务链路之外的运营前置，全部走真实表）：
	//   registration_enabled=true       —— 默认关闭注册，必须显式开启
	//   openai 标准分组 + 直通账号 + 绑定 —— 网关调度所需的最小渠道配置
	// ---------------------------------------------------------------
	_, err = entClient.Setting.Create().
		SetKey("registration_enabled").
		SetValue("true").
		Save(ctx)
	require.NoError(t, err, "开启注册开关")

	group, err := entClient.Group.Create().
		SetName("e2e-critical-path-standard").
		SetPlatform("openai").
		SetStatus("active").
		SetSubscriptionType("standard").
		SetRateMultiplier(1.0).
		SetIsExclusive(false).
		Save(ctx)
	require.NoError(t, err, "创建标准分组")

	account, err := entClient.Account.Create().
		SetName("e2e-critical-path-upstream").
		SetPlatform("openai").
		SetType("apikey").
		SetCredentials(map[string]any{
			"api_key":       "sk-fake-upstream-credential",
			"base_url":      upstream.URL,
			"model_mapping": map[string]any{cpModel: cpModel},
		}).
		// openai_passthrough=true：chat/completions 原样直转到 base_url，
		// 假上游只需实现 OpenAI 非流式 JSON 响应。
		SetExtra(map[string]any{"openai_passthrough": true}).
		SetStatus("active").
		SetSchedulable(true).
		SetPriority(1).
		SetConcurrency(3).
		SetRateMultiplier(1.0).
		Save(ctx)
	require.NoError(t, err, "创建上游账号")

	_, err = entClient.AccountGroup.Create().
		SetAccountID(account.ID).
		SetGroupID(group.ID).
		SetPriority(1).
		Save(ctx)
	require.NoError(t, err, "绑定账号到分组")

	// ---------------------------------------------------------------
	// 编译并启动真实 server 二进制
	// ---------------------------------------------------------------
	wd, err := os.Getwd()
	require.NoError(t, err)
	backendRoot := filepath.Clean(filepath.Join(wd, "..", ".."))

	exeSuffix := ""
	if runtime.GOOS == "windows" {
		exeSuffix = ".exe"
	}
	serverBin := filepath.Join(t.TempDir(), "sub2api-e2e-server"+exeSuffix)

	buildCtx, cancelBuild := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelBuild()
	buildCmd := exec.CommandContext(buildCtx, "go", "build", "-o", serverBin, "./cmd/server")
	buildCmd.Dir = backendRoot
	buildOut, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "编译 server 二进制失败：\n%s", string(buildOut))

	pgHost, err := pgContainer.Host(ctx)
	require.NoError(t, err)
	pgPort, err := pgContainer.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)
	redisPort, err := redisContainer.MappedPort(ctx, "6379/tcp")
	require.NoError(t, err)

	serverPort := cpFreePort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", serverPort)

	logPath := filepath.Join(t.TempDir(), "server.log")
	logFile, err := os.Create(logPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = logFile.Close() })

	serverCmd := exec.Command(serverBin)
	// CWD 指向 backend 根目录，保证 ./resources/model-pricing 等相对路径可用。
	serverCmd.Dir = backendRoot
	serverCmd.Stdout = logFile
	serverCmd.Stderr = logFile
	serverCmd.Env = append(os.Environ(),
		"DATABASE_HOST="+pgHost,
		fmt.Sprintf("DATABASE_PORT=%d", pgPort.Int()),
		"DATABASE_USER=postgres",
		"DATABASE_PASSWORD=postgres",
		"DATABASE_DBNAME=sub2api_e2e",
		"DATABASE_SSLMODE=disable",
		"REDIS_HOST="+redisHost,
		fmt.Sprintf("REDIS_PORT=%d", redisPort.Int()),
		"JWT_SECRET="+cpJWTSecret,
		"SERVER_HOST=127.0.0.1",
		fmt.Sprintf("SERVER_PORT=%d", serverPort),
	)
	require.NoError(t, serverCmd.Start(), "启动 server 进程")
	t.Cleanup(func() {
		_ = serverCmd.Process.Kill()
		_, _ = serverCmd.Process.Wait()
	})

	cpWaitHealthy(t, baseURL, logPath, 90*time.Second)

	httpClient := &http.Client{Timeout: 60 * time.Second}

	// ---------------------------------------------------------------
	// 步骤 1：注册新用户（注册成功直接返回 token，无需单独登录）
	// ---------------------------------------------------------------
	email := fmt.Sprintf("e2e-critical-%d@test.local", time.Now().UnixNano())
	registerBody := map[string]string{
		"email":    email,
		"password": "E2eCriticalPath!123",
	}
	var auth cpAuthData
	cpPostJSON(t, httpClient, baseURL+"/api/v1/auth/register", "", registerBody, &auth)
	require.NotEmpty(t, auth.AccessToken, "注册应返回 access_token")
	require.Greater(t, auth.User.ID, int64(0), "注册应返回用户 ID")
	t.Logf("步骤 1 通过：注册用户 %s (id=%d)", email, auth.User.ID)

	// 网关鉴权与计费预检要求 balance > 0（新用户默认 0），属于运营前置而非被测链路，
	// 直接在真实表上充值。
	require.NoError(t,
		entClient.User.UpdateOneID(auth.User.ID).SetBalance(100).Exec(ctx),
		"为测试用户充值余额")

	// ---------------------------------------------------------------
	// 步骤 2：创建 API Key（本分支要求必须绑定分组）
	// ---------------------------------------------------------------
	createKeyBody := map[string]any{
		"name":     "e2e-critical-path-key",
		"group_id": group.ID,
	}
	var apiKey cpAPIKeyData
	cpPostJSON(t, httpClient, baseURL+"/api/v1/keys", auth.AccessToken, createKeyBody, &apiKey)
	require.NotEmpty(t, apiKey.Key, "创建 Key 应返回完整明文 key")
	require.Greater(t, apiKey.ID, int64(0))
	t.Logf("步骤 2 通过：创建 API Key id=%d", apiKey.ID)

	// ---------------------------------------------------------------
	// 步骤 3：用该 Key 调用 GET /v1/models，必须 200
	// ---------------------------------------------------------------
	modelsReq, err := http.NewRequest(http.MethodGet, baseURL+"/v1/models", nil)
	require.NoError(t, err)
	modelsReq.Header.Set("Authorization", "Bearer "+apiKey.Key)
	modelsResp, err := httpClient.Do(modelsReq)
	require.NoError(t, err)
	modelsBody, _ := io.ReadAll(modelsResp.Body)
	_ = modelsResp.Body.Close()
	require.Equal(t, http.StatusOK, modelsResp.StatusCode,
		"GET /v1/models 应返回 200，实际响应：%s", string(modelsBody))
	t.Logf("步骤 3 通过：GET /v1/models -> 200，body=%s", string(modelsBody))

	// ---------------------------------------------------------------
	// 步骤 4：用该 Key 调用 POST /v1/chat/completions，必须 200 且 usage 不为空
	// ---------------------------------------------------------------
	chatPayload, err := json.Marshal(map[string]any{
		"model":      cpModel,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"stream":     false,
		"max_tokens": 64,
	})
	require.NoError(t, err)
	chatReq, err := http.NewRequest(http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(chatPayload))
	require.NoError(t, err)
	chatReq.Header.Set("Content-Type", "application/json")
	chatReq.Header.Set("Authorization", "Bearer "+apiKey.Key)
	chatResp, err := httpClient.Do(chatReq)
	require.NoError(t, err)
	chatBody, _ := io.ReadAll(chatResp.Body)
	_ = chatResp.Body.Close()
	require.Equal(t, http.StatusOK, chatResp.StatusCode,
		"POST /v1/chat/completions 应返回 200，实际响应：%s", string(chatBody))

	var chat cpChatCompletionResponse
	require.NoError(t, json.Unmarshal(chatBody, &chat), "chat completions 响应应为合法 JSON：%s", string(chatBody))
	require.Greater(t, chat.Usage.PromptTokens, 0, "usage.prompt_tokens 不应为空")
	require.Greater(t, chat.Usage.CompletionTokens, 0, "usage.completion_tokens 不应为空")
	t.Logf("步骤 4 通过：chat/completions -> 200, usage=%+v", chat.Usage)

	// ---------------------------------------------------------------
	// 步骤 5：usage_logs 有对应记录且 actual_cost > 0
	// （写入走异步 worker pool，无 flush 钩子，只能轮询等待）
	// ---------------------------------------------------------------
	var logged *dbent.UsageLog
	require.Eventually(t, func() bool {
		found, qErr := entClient.UsageLog.Query().
			Where(dbusagelog.APIKeyID(apiKey.ID)).
			First(ctx)
		if qErr != nil {
			return false
		}
		logged = found
		return true
	}, 30*time.Second, 500*time.Millisecond, "等待 usage_logs 异步落库超时")

	require.Equal(t, auth.User.ID, logged.UserID, "usage_logs.user_id 应为注册用户")
	require.Equal(t, cpModel, logged.Model, "usage_logs.model 应为请求模型")
	require.Greater(t, logged.ActualCost, 0.0,
		"usage_logs.actual_cost 应大于 0（tokens=%d/%d, total_cost=%v）",
		logged.InputTokens, logged.OutputTokens, logged.TotalCost)
	t.Logf("步骤 5 通过：usage_logs id=%d actual_cost=%v", logged.ID, logged.ActualCost)
}

// cpPostJSON 发送 JSON POST，要求 HTTP 200 + 信封 code==0，并把 data 解到 out。
func cpPostJSON(t *testing.T, client *http.Client, url, bearer string, body any, out any) {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	resp, err := client.Do(req)
	require.NoError(t, err, "POST %s", url)
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "POST %s 响应：%s", url, string(raw))

	var envelope cpEnvelope
	require.NoError(t, json.Unmarshal(raw, &envelope), "POST %s 响应应为信封 JSON：%s", url, string(raw))
	require.Equal(t, 0, envelope.Code, "POST %s 业务码应为 0：%s", url, string(raw))
	if out != nil {
		require.NoError(t, json.Unmarshal(envelope.Data, out), "POST %s data 解析失败：%s", url, string(raw))
	}
}

func cpOpenSQL(t *testing.T, ctx context.Context, dsn string) *sql.DB {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		db, err := sql.Open("postgres", dsn)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
			err = db.PingContext(pingCtx)
			cancel()
			if err == nil {
				t.Cleanup(func() { _ = db.Close() })
				return db
			}
			_ = db.Close()
		}
		lastErr = err
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("数据库 30s 内未就绪: %v", lastErr)
	return nil
}

func cpFreePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	return port
}

func cpWaitHealthy(t *testing.T, baseURL, logPath string, timeout time.Duration) {
	t.Helper()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	serverLog, _ := os.ReadFile(logPath)
	t.Fatalf("server 在 %s 内未就绪，日志：\n%s", timeout, string(serverLog))
}
