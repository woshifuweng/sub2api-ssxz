package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func (s *AntigravityGatewayService) ForwardUpstreamContext(ctx context.Context, c gatewayctx.GatewayContext, account *Account, body []byte) (*ForwardResult, error) {
	startTime := time.Now()
	sessionID := getSessionIDContext(c)
	prefix := logPrefix(sessionID, account.Name)

	// 获取上游配置
	baseURL := strings.TrimSpace(account.GetCredential("base_url"))
	apiKey := strings.TrimSpace(account.GetCredential("api_key"))
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("upstream account missing base_url or api_key")
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	// 解析请求获取模型信息
	var claudeReq antigravity.ClaudeRequest
	if err := json.Unmarshal(body, &claudeReq); err != nil {
		return nil, fmt.Errorf("parse claude request: %w", err)
	}
	if strings.TrimSpace(claudeReq.Model) == "" {
		return nil, fmt.Errorf("missing model")
	}
	originalModel := claudeReq.Model

	// 构建上游请求 URL
	upstreamURL := baseURL + "/v1/messages"

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("x-api-key", apiKey) // Claude API 兼容

	// 透传 Claude 相关 headers
	if v := c.HeaderValue("anthropic-version"); v != "" {
		req.Header.Set("anthropic-version", v)
	}
	if v := c.HeaderValue("anthropic-beta"); v != "" {
		req.Header.Set("anthropic-beta", v)
	}

	// 代理 URL
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 发送请求
	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "%s upstream request failed: %v", prefix, err)
		return nil, fmt.Errorf("upstream request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 处理错误响应
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

		// 429 错误时标记账号限流
		if resp.StatusCode == http.StatusTooManyRequests {
			s.handleUpstreamError(ctx, prefix, account, resp.StatusCode, resp.Header, respBody, originalModel, 0, "", false)
		}

		// 透传上游错误
		c.SetHeader("Content-Type", resp.Header.Get("Content-Type"))
		_, _ = c.WriteBytes(resp.StatusCode, respBody)

		return &ForwardResult{
			Model: originalModel,
		}, nil
	}

	// 处理成功响应（流式/非流式）
	var usage *ClaudeUsage
	var firstTokenMs *int
	var clientDisconnect bool

	if claudeReq.Stream {
		// 流式响应：透传
		gatewayctx.PrepareSSE(c, gatewayctx.SSEOptions{
			ContentType:  "text/event-stream",
			CacheControl: "no-cache, no-transform",
		})

		streamRes := s.streamUpstreamResponseContext(c, resp, startTime)
		usage = streamRes.usage
		firstTokenMs = streamRes.firstTokenMs
		clientDisconnect = streamRes.clientDisconnect
	} else {
		// 非流式响应：直接透传
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read upstream response: %w", err)
		}

		// 提取 usage
		usage = s.extractClaudeUsage(respBody)

		c.SetHeader("Content-Type", resp.Header.Get("Content-Type"))
		_, _ = c.WriteBytes(http.StatusOK, respBody)
	}

	// 构建计费结果
	duration := time.Since(startTime)
	logger.LegacyPrintf("service.antigravity_gateway", "%s status=success duration_ms=%d", prefix, duration.Milliseconds())

	return &ForwardResult{
		Model:            originalModel,
		Stream:           claudeReq.Stream,
		Duration:         duration,
		FirstTokenMs:     firstTokenMs,
		ClientDisconnect: clientDisconnect,
		Usage: ClaudeUsage{
			InputTokens:              usage.InputTokens,
			OutputTokens:             usage.OutputTokens,
			CacheReadInputTokens:     usage.CacheReadInputTokens,
			CacheCreationInputTokens: usage.CacheCreationInputTokens,
		},
	}, nil
}

// streamUpstreamResponse 透传上游 SSE 流并提取 Claude usage

func (s *AntigravityGatewayService) streamUpstreamResponseContext(c gatewayctx.GatewayContext, resp *http.Response, startTime time.Time) *antigravityStreamResult {
	usage := &ClaudeUsage{}
	var firstTokenMs *int

	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanner.Buffer(make([]byte, 64*1024), maxLineSize)

	type scanEvent struct {
		line string
		err  error
	}
	events := make(chan scanEvent, 16)
	done := make(chan struct{})
	sendEvent := func(ev scanEvent) bool {
		select {
		case events <- ev:
			return true
		case <-done:
			return false
		}
	}
	var lastReadAt int64
	atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
	go func() {
		defer close(events)
		for scanner.Scan() {
			atomic.StoreInt64(&lastReadAt, time.Now().UnixNano())
			if !sendEvent(scanEvent{line: scanner.Text()}) {
				return
			}
		}
		if err := scanner.Err(); err != nil {
			_ = sendEvent(scanEvent{err: err})
		}
	}()
	defer close(done)

	streamInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamDataIntervalTimeout > 0 {
		streamInterval = time.Duration(s.settingService.cfg.Gateway.StreamDataIntervalTimeout) * time.Second
	}
	var intervalTicker *time.Ticker
	if streamInterval > 0 {
		intervalTicker = time.NewTicker(streamInterval)
		defer intervalTicker.Stop()
	}
	var intervalCh <-chan time.Time
	if intervalTicker != nil {
		intervalCh = intervalTicker.C
	}

	// 下游 keepalive：防止代理/Cloudflare Tunnel 因连接空闲而断开
	keepaliveInterval := time.Duration(0)
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.StreamKeepaliveInterval > 0 {
		keepaliveInterval = time.Duration(s.settingService.cfg.Gateway.StreamKeepaliveInterval) * time.Second
	}
	var keepaliveTicker *time.Ticker
	if keepaliveInterval > 0 {
		keepaliveTicker = time.NewTicker(keepaliveInterval)
		defer keepaliveTicker.Stop()
	}
	var keepaliveCh <-chan time.Time
	if keepaliveTicker != nil {
		keepaliveCh = keepaliveTicker.C
	}
	lastDataAt := time.Now()

	cw := newAntigravityGatewayWriter(c, "antigravity upstream")

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: cw.Disconnected()}
			}
			if ev.err != nil {
				if disconnect, handled := handleStreamReadError(ev.err, cw.Disconnected(), "antigravity upstream"); handled {
					return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: disconnect}
				}
				logger.LegacyPrintf("service.antigravity_gateway", "Stream read error (antigravity upstream): %v", ev.err)
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}
			}

			lastDataAt = time.Now()

			line := ev.line

			// 记录首 token 时间
			if firstTokenMs == nil && len(line) > 0 {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}

			// 尝试从 message_delta 或 message_stop 事件提取 usage
			s.extractSSEUsage(line, usage)

			// 透传行
			cw.Fprintf("%s\n", line)

		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			if cw.Disconnected() {
				logger.LegacyPrintf("service.antigravity_gateway", "Upstream timeout after client disconnect (antigravity upstream), returning collected usage")
				return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs, clientDisconnect: true}
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity upstream)")
			return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}

		case <-keepaliveCh:
			if cw.Disconnected() {
				continue
			}
			if time.Since(lastDataAt) < keepaliveInterval {
				continue
			}
			// SSE ping 事件：Anthropic 原生格式，客户端会正确处理，
			// 同时保持连接活跃防止 Cloudflare Tunnel 等代理断开
			if !cw.Fprintf("event: ping\ndata: {\"type\": \"ping\"}\n\n") {
				logger.LegacyPrintf("service.antigravity_gateway", "Client disconnected during keepalive ping (antigravity upstream), continuing to drain upstream for billing")
				continue
			}
		}
	}
}

// extractSSEUsage 从 SSE data 行中提取 Claude usage（用于流式透传场景）
