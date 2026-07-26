package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// ---------- fakes ----------

// channelMonitorUserSettingRepo 只实现 GetMultiple，供 SettingService.GetChannelMonitorRuntime 读开关。
// 其余方法由嵌入接口占位；一旦被调用会 panic，从而暴露非预期依赖。
type channelMonitorUserSettingRepo struct {
	service.SettingRepository
	values map[string]string
}

func (r *channelMonitorUserSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	out := make(map[string]string, len(r.values))
	for k, v := range r.values {
		out[k] = v
	}
	return out, nil
}

// channelMonitorUserGroups 伪造用户可见分组（channelMonitorGroupAccess）。
type channelMonitorUserGroups struct {
	groups []service.Group
	calls  int
}

func (g *channelMonitorUserGroups) GetAvailableGroups(context.Context, int64) ([]service.Group, error) {
	g.calls++
	return g.groups, nil
}

// channelMonitorUserRepo 只实现用户视图聚合路径用到的方法。
// 未实现的方法保持嵌入接口的 nil 实现：若用户路径意外调用（例如误用 admin 查询），测试会 panic。
type channelMonitorUserRepo struct {
	service.ChannelMonitorRepository
	monitors []*service.ChannelMonitor
	latest   []*service.ChannelMonitorLatest
	avail    []*service.ChannelMonitorAvailability
	history  []*service.ChannelMonitorHistoryEntry
	listed   int
}

func (r *channelMonitorUserRepo) ListEnabled(context.Context) ([]*service.ChannelMonitor, error) {
	r.listed++
	return r.monitors, nil
}

func (r *channelMonitorUserRepo) GetByID(_ context.Context, id int64) (*service.ChannelMonitor, error) {
	for _, m := range r.monitors {
		if m.ID == id {
			return m, nil
		}
	}
	return nil, service.ErrChannelMonitorNotFound
}

func (r *channelMonitorUserRepo) ListLatestForMonitorIDs(_ context.Context, ids []int64) (map[int64][]*service.ChannelMonitorLatest, error) {
	out := make(map[int64][]*service.ChannelMonitorLatest, len(ids))
	for _, id := range ids {
		out[id] = r.latest
	}
	return out, nil
}

func (r *channelMonitorUserRepo) ComputeAvailabilityForMonitors(_ context.Context, ids []int64, windowDays int) (map[int64][]*service.ChannelMonitorAvailability, error) {
	out := make(map[int64][]*service.ChannelMonitorAvailability, len(ids))
	for _, id := range ids {
		out[id] = r.availFor(windowDays)
	}
	return out, nil
}

func (r *channelMonitorUserRepo) ListRecentHistoryForMonitors(_ context.Context, ids []int64, _ map[int64]string, _ int) (map[int64][]*service.ChannelMonitorHistoryEntry, error) {
	out := make(map[int64][]*service.ChannelMonitorHistoryEntry, len(ids))
	for _, id := range ids {
		out[id] = r.history
	}
	return out, nil
}

func (r *channelMonitorUserRepo) ListLatestPerModel(context.Context, int64) ([]*service.ChannelMonitorLatest, error) {
	return r.latest, nil
}

func (r *channelMonitorUserRepo) ComputeAvailability(_ context.Context, _ int64, windowDays int) ([]*service.ChannelMonitorAvailability, error) {
	return r.availFor(windowDays), nil
}

func (r *channelMonitorUserRepo) availFor(windowDays int) []*service.ChannelMonitorAvailability {
	out := make([]*service.ChannelMonitorAvailability, 0, len(r.avail))
	for _, a := range r.avail {
		clone := *a
		clone.WindowDays = windowDays
		out = append(out, &clone)
	}
	return out
}

// 敏感值：这些字符串一旦出现在用户端响应里即为泄露。
const (
	secretAPIKey     = "sk-upstream-secret-key-xyz"
	secretEndpoint   = "https://internal-upstream.example.net/v1"
	secretHeaderTok  = "tok-upstream-header-abc123"
	secretErrDetail  = "401 Unauthorized: invalid api key sk-upstream-secret-key-xyz"
	visibleGroupName = "vip"
	hiddenGroupName  = "internal-only"
)

func channelMonitorUserFixtureRepo() *channelMonitorUserRepo {
	checkedAt := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	latency := 320
	ping := 45
	return &channelMonitorUserRepo{
		monitors: []*service.ChannelMonitor{
			{
				ID:                   7,
				Name:                 "Claude 高速通道",
				Provider:             "anthropic",
				APIMode:              "messages",
				Endpoint:             secretEndpoint,
				APIKey:               secretAPIKey,
				PrimaryModel:         "claude-sonnet-4-6",
				ExtraModels:          []string{"claude-haiku-4-5"},
				GroupName:            visibleGroupName,
				Enabled:              true,
				IntervalSeconds:      60,
				JitterSeconds:        5,
				LastCheckedAt:        &checkedAt,
				CreatedBy:            9001,
				TemplateID:           channelMonitorUserInt64Ptr(55),
				ExtraHeaders:         map[string]string{"X-Upstream-Token": secretHeaderTok},
				BodyOverrideMode:     "merge",
				BodyOverride:         map[string]any{"upstream_account": "acct-77"},
				DuplicateOperationID: "dup-op-1",
			},
			{
				// 用户不可见分组：必须被 ListUserView 过滤掉。
				ID:           8,
				Name:         "内部压测渠道",
				Provider:     "openai",
				Endpoint:     secretEndpoint,
				APIKey:       secretAPIKey,
				PrimaryModel: "gpt-4o",
				GroupName:    hiddenGroupName,
				Enabled:      true,
			},
		},
		latest: []*service.ChannelMonitorLatest{
			{Model: "claude-sonnet-4-6", Status: "operational", LatencyMs: &latency, PingLatencyMs: &ping, CheckedAt: checkedAt},
			{Model: "claude-haiku-4-5", Status: "degraded", LatencyMs: &latency, CheckedAt: checkedAt},
		},
		avail: []*service.ChannelMonitorAvailability{
			{Model: "claude-sonnet-4-6", TotalChecks: 100, OperationalChecks: 99, AvailabilityPct: 99, AvgLatencyMs: &latency},
			{Model: "claude-haiku-4-5", TotalChecks: 100, OperationalChecks: 90, AvailabilityPct: 90, AvgLatencyMs: &latency},
		},
		history: []*service.ChannelMonitorHistoryEntry{
			// Message 携带上游凭证：timeline 必须剥掉 message。
			{ID: 1, Model: "claude-sonnet-4-6", Status: "operational", LatencyMs: &latency, PingLatencyMs: &ping, Message: secretErrDetail, CheckedAt: checkedAt},
			{ID: 2, Model: "claude-sonnet-4-6", Status: "failed", Message: secretErrDetail, CheckedAt: checkedAt.Add(-time.Minute)},
		},
	}
}

func channelMonitorUserInt64Ptr(v int64) *int64 { return &v }

// newChannelMonitorUserHandler 组装 handler：settingValues 为 nil 表示不接 SettingService（视为开关开启）。
func newChannelMonitorUserHandler(
	repo *channelMonitorUserRepo,
	groups *channelMonitorUserGroups,
	settingValues map[string]string,
) *ChannelMonitorUserHandler {
	h := &ChannelMonitorUserHandler{}
	if repo != nil {
		h.monitorService = service.NewChannelMonitorService(repo, nil)
	}
	if groups != nil {
		h.apiKeyService = groups
	}
	if settingValues != nil {
		h.settingService = service.NewSettingService(&channelMonitorUserSettingRepo{values: settingValues}, nil)
	}
	return h
}

func newChannelMonitorUserRequest(t *testing.T, method, target string, userID int64, params gin.Params) (*httptest.ResponseRecorder, *gin.Context) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	c.Params = params
	if userID > 0 {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: userID, Concurrency: 1})
	}
	return w, c
}

// ---------- 1. 开关关闭 → 拒绝 ----------

// 开关关闭时 List 必须返回空集合，且不得触达 monitor / 分组依赖，
// 与前端路由守卫（channel_monitor_enabled=false 时不进页面）保持一致。
func TestChannelMonitorUser_FeatureDisabled_ListReturnsEmptyAndSkipsService(t *testing.T) {
	repo := channelMonitorUserFixtureRepo()
	groups := &channelMonitorUserGroups{groups: []service.Group{{ID: 1, Name: visibleGroupName}}}
	h := newChannelMonitorUserHandler(repo, groups, map[string]string{
		service.SettingKeyChannelMonitorEnabled: "false",
	})

	w, c := newChannelMonitorUserRequest(t, http.MethodGet, "/api/v1/channel-monitors", 42, nil)
	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, 0, repo.listed, "开关关闭时不应查询监控列表")
	require.Equal(t, 0, groups.calls, "开关关闭时不应解析用户可见分组")

	var body struct {
		Code int `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	require.Empty(t, body.Data.Items)
	// 空集合必须序列化为 []，不能是 null，否则前端 res.items 兜底逻辑会拿到 null。
	require.Contains(t, w.Body.String(), `"items":[]`)
}

// 开关关闭时详情接口必须按"资源不存在"处理（404），不泄露监控是否存在。
func TestChannelMonitorUser_FeatureDisabled_StatusReturns404(t *testing.T) {
	repo := channelMonitorUserFixtureRepo()
	groups := &channelMonitorUserGroups{groups: []service.Group{{ID: 1, Name: visibleGroupName}}}
	h := newChannelMonitorUserHandler(repo, groups, map[string]string{
		service.SettingKeyChannelMonitorEnabled: "false",
	})

	w, c := newChannelMonitorUserRequest(t, http.MethodGet, "/api/v1/channel-monitors/7/status", 42,
		gin.Params{{Key: "id", Value: "7"}})
	h.GetStatus(c)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, 0, groups.calls, "开关关闭时不应解析用户可见分组")
	require.NotContains(t, w.Body.String(), secretEndpoint)
}

// ---------- 2. 未登录 → 401 ----------

// 未注入 AuthSubject（未登录）时两个接口都必须 401，且不触达 service 依赖。
func TestChannelMonitorUser_Unauthenticated401(t *testing.T) {
	cases := []struct {
		name   string
		target string
		params gin.Params
		invoke func(h *ChannelMonitorUserHandler, c *gin.Context)
	}{
		{
			name:   "list",
			target: "/api/v1/channel-monitors",
			invoke: func(h *ChannelMonitorUserHandler, c *gin.Context) { h.List(c) },
		},
		{
			name:   "status",
			target: "/api/v1/channel-monitors/7/status",
			params: gin.Params{{Key: "id", Value: "7"}},
			invoke: func(h *ChannelMonitorUserHandler, c *gin.Context) { h.GetStatus(c) },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := channelMonitorUserFixtureRepo()
			groups := &channelMonitorUserGroups{}
			// settingValues 传 nil：开关视为开启，确保 401 来自鉴权而非功能开关。
			h := newChannelMonitorUserHandler(repo, groups, nil)

			w, c := newChannelMonitorUserRequest(t, http.MethodGet, tc.target, 0, tc.params)
			tc.invoke(h, c)

			require.Equal(t, http.StatusUnauthorized, w.Code)
			require.Equal(t, 0, repo.listed)
			require.Equal(t, 0, groups.calls)
		})
	}
}

// ---------- 3. 返回体不含敏感字段 ----------

// channelMonitorUserSecrets 汇总所有绝不允许出现在用户端响应中的字符串。
func channelMonitorUserSecrets() []string {
	return []string{
		secretAPIKey,    // 上游 API key
		secretEndpoint,  // 上游真实 URL
		secretHeaderTok, // 自定义上游鉴权头
		secretErrDetail, // 含凭证的错误详情
		hiddenGroupName, // 用户无权访问的分组
		"内部压测渠道",        // 不可见渠道的展示名
		"acct-77",       // 上游账号 ID
		"dup-op-1",      // 内部复制操作 ID
		"9001",          // created_by 管理员 ID
	}
}

// channelMonitorUserForbiddenKeys 管理端字段：用户端 DTO 一律不得出现。
func channelMonitorUserForbiddenKeys() []string {
	return []string{
		"api_key", "apikey", "endpoint", "base_url", "api_mode",
		"extra_headers", "body_override", "body_override_mode",
		"created_by", "template_id", "duplicate_operation_id",
		"interval_seconds", "jitter_seconds", "last_checked_at",
		"created_at", "updated_at", "enabled",
		"api_key_decrypt_failed",
	}
	// 注意：不校验 "message" —— 统一响应信封本身有 message 字段。
	// timeline / model DTO 不含 message 由各自的字段数断言 + secretErrDetail 检查覆盖。
}

func requireNoChannelMonitorSecrets(t *testing.T, raw string) {
	t.Helper()
	for _, secret := range channelMonitorUserSecrets() {
		require.NotContainsf(t, raw, secret, "用户端响应泄露敏感值 %q", secret)
	}
	lower := strings.ToLower(raw)
	for _, key := range channelMonitorUserForbiddenKeys() {
		require.NotContainsf(t, lower, `"`+key+`"`, "用户端响应包含管理端字段 %q", key)
	}
}

// List 响应只暴露展示名/模型/状态/延迟/可用率/时间桶，且按用户可见分组过滤。
func TestChannelMonitorUser_ListResponseExcludesSensitiveFields(t *testing.T) {
	repo := channelMonitorUserFixtureRepo()
	groups := &channelMonitorUserGroups{groups: []service.Group{{ID: 1, Name: visibleGroupName}}}
	h := newChannelMonitorUserHandler(repo, groups, map[string]string{
		service.SettingKeyChannelMonitorEnabled: "true",
	})

	w, c := newChannelMonitorUserRequest(t, http.MethodGet, "/api/v1/channel-monitors", 42, nil)
	h.List(c)

	require.Equal(t, http.StatusOK, w.Code)
	requireNoChannelMonitorSecrets(t, w.Body.String())

	var body struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	// 只有 vip 分组那条可见；internal-only 分组被过滤。
	require.Len(t, body.Data.Items, 1)

	item := body.Data.Items[0]
	for _, key := range []string{
		"id", "name", "provider", "group_name", "primary_model", "primary_status",
		"primary_latency_ms", "primary_ping_latency_ms", "availability_7d",
		"extra_models", "timeline",
	} {
		require.Containsf(t, item, key, "用户端 list DTO 缺少字段 %q", key)
	}
	require.Len(t, item, 11, "用户端 list DTO 字段集合发生变化，请复核脱敏")
	require.Equal(t, "Claude 高速通道", item["name"])

	// timeline 点只保留状态/延迟/时间，不含 message。
	timeline, ok := item["timeline"].([]any)
	require.True(t, ok)
	require.Len(t, timeline, 2)
	point, ok := timeline[0].(map[string]any)
	require.True(t, ok)
	require.Len(t, point, 4)
	for _, key := range []string{"status", "latency_ms", "ping_latency_ms", "checked_at"} {
		require.Containsf(t, point, key, "timeline 点缺少字段 %q", key)
	}

	// extra_models 只保留 model/status/latency_ms。
	extras, ok := item["extra_models"].([]any)
	require.True(t, ok)
	require.Len(t, extras, 1)
	extra, ok := extras[0].(map[string]any)
	require.True(t, ok)
	require.Len(t, extra, 3)
	require.Equal(t, "claude-haiku-4-5", extra["model"])
}

// 详情响应只暴露每模型的可用率/延迟窗口统计。
func TestChannelMonitorUser_StatusResponseExcludesSensitiveFields(t *testing.T) {
	repo := channelMonitorUserFixtureRepo()
	groups := &channelMonitorUserGroups{groups: []service.Group{{ID: 1, Name: visibleGroupName}}}
	h := newChannelMonitorUserHandler(repo, groups, map[string]string{
		service.SettingKeyChannelMonitorEnabled: "true",
	})

	w, c := newChannelMonitorUserRequest(t, http.MethodGet, "/api/v1/channel-monitors/7/status", 42,
		gin.Params{{Key: "id", Value: "7"}})
	h.GetStatus(c)

	require.Equal(t, http.StatusOK, w.Code)
	requireNoChannelMonitorSecrets(t, w.Body.String())

	var body struct {
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	for _, key := range []string{"id", "name", "provider", "group_name", "models"} {
		require.Containsf(t, body.Data, key, "用户端 detail DTO 缺少字段 %q", key)
	}
	require.Len(t, body.Data, 5, "用户端 detail DTO 字段集合发生变化，请复核脱敏")

	models, ok := body.Data["models"].([]any)
	require.True(t, ok)
	require.Len(t, models, 2)
	model, ok := models[0].(map[string]any)
	require.True(t, ok)
	for _, key := range []string{
		"model", "latest_status", "latest_latency_ms",
		"availability_7d", "availability_15d", "availability_30d", "avg_latency_7d_ms",
	} {
		require.Containsf(t, model, key, "用户端 model DTO 缺少字段 %q", key)
	}
	require.Len(t, model, 7, "用户端 model DTO 字段集合发生变化，请复核脱敏")
}

// 用户可见分组不含该监控所属分组时，详情按 404 处理（越权探测防护）。
func TestChannelMonitorUser_StatusHiddenGroupReturns404(t *testing.T) {
	repo := channelMonitorUserFixtureRepo()
	groups := &channelMonitorUserGroups{groups: []service.Group{{ID: 1, Name: visibleGroupName}}}
	h := newChannelMonitorUserHandler(repo, groups, map[string]string{
		service.SettingKeyChannelMonitorEnabled: "true",
	})

	// monitor 8 属于 internal-only 分组，用户只有 vip。
	w, c := newChannelMonitorUserRequest(t, http.MethodGet, "/api/v1/channel-monitors/8/status", 42,
		gin.Params{{Key: "id", Value: "8"}})
	h.GetStatus(c)

	require.Equal(t, http.StatusNotFound, w.Code)
	requireNoChannelMonitorSecrets(t, w.Body.String())
}
