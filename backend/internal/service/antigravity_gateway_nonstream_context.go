package service

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
)

func (s *AntigravityGatewayService) handleClaudeStreamToNonStreamingContext(c gatewayctx.GatewayContext, resp *http.Response, startTime time.Time, originalModel string) (*antigravityStreamResult, error) {
	scanner := bufio.NewScanner(resp.Body)
	maxLineSize := defaultMaxLineSize
	if s.settingService.cfg != nil && s.settingService.cfg.Gateway.MaxLineSize > 0 {
		maxLineSize = s.settingService.cfg.Gateway.MaxLineSize
	}
	scanBuf := getSSEScannerBuf64K()
	scanner.Buffer(scanBuf[:0], maxLineSize)

	var firstTokenMs *int
	var last map[string]any
	var lastWithParts map[string]any
	var collectedParts []map[string]any // 收集所有 parts（包括 text、thinking、functionCall、inlineData 等）

	type scanEvent struct {
		line string
		err  error
	}

	// 独立 goroutine 读取上游，避免读取阻塞影响超时处理
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
	go func(scanBuf *sseScannerBuf64K) {
		defer putSSEScannerBuf64K(scanBuf)
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
	}(scanBuf)
	defer close(done)

	// 上游数据间隔超时保护（防止上游挂起长期占用连接）
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

	for {
		select {
		case ev, ok := <-events:
			if !ok {
				// 流结束，转换并返回响应
				goto returnResponse
			}
			if ev.err != nil {
				if errors.Is(ev.err, bufio.ErrTooLong) {
					logger.LegacyPrintf("service.antigravity_gateway", "SSE line too long (antigravity claude non-stream): max_size=%d error=%v", maxLineSize, ev.err)
				}
				return nil, ev.err
			}

			line := ev.line
			trimmed := strings.TrimRight(line, "\r\n")

			if !strings.HasPrefix(trimmed, "data:") {
				continue
			}

			payload := strings.TrimSpace(strings.TrimPrefix(trimmed, "data:"))
			if payload == "" || payload == "[DONE]" {
				continue
			}

			// 解包 v1internal 响应
			inner, parseErr := s.unwrapV1InternalResponse([]byte(payload))
			if parseErr != nil {
				continue
			}

			var parsed map[string]any
			if err := json.Unmarshal(inner, &parsed); err != nil {
				continue
			}

			// 记录首 token 时间
			if firstTokenMs == nil {
				ms := int(time.Since(startTime).Milliseconds())
				firstTokenMs = &ms
			}

			last = parsed

			// 保留最后一个有 parts 的响应，并收集所有 parts
			if parts := extractGeminiParts(parsed); len(parts) > 0 {
				lastWithParts = parsed

				// 收集所有 parts（text、thinking、functionCall、inlineData 等）
				collectedParts = append(collectedParts, parts...)
			}

		case <-intervalCh:
			lastRead := time.Unix(0, atomic.LoadInt64(&lastReadAt))
			if time.Since(lastRead) < streamInterval {
				continue
			}
			logger.LegacyPrintf("service.antigravity_gateway", "Stream data interval timeout (antigravity claude non-stream)")
			return nil, fmt.Errorf("stream data interval timeout")
		}
	}

returnResponse:
	// 选择最后一个有效响应
	finalResponse := pickGeminiCollectResult(last, lastWithParts)

	// 处理空响应情况 — 触发同账号重试 + failover 切换账号
	if last == nil && lastWithParts == nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] warning: empty stream response (claude non-stream), triggering failover")
		return nil, &UpstreamFailoverError{
			StatusCode:             http.StatusBadGateway,
			ResponseBody:           []byte(`{"error":"empty stream response from upstream"}`),
			RetryableOnSameAccount: true,
		}
	}

	// 将收集的所有 parts 合并到最终响应中
	if len(collectedParts) > 0 {
		finalResponse = mergeCollectedPartsToResponse(finalResponse, collectedParts)
	}

	// 序列化为 JSON（Gemini 格式）
	geminiBody, err := json.Marshal(finalResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal gemini response: %w", err)
	}

	// 转换 Gemini 响应为 Claude 格式
	claudeResp, agUsage, err := antigravity.TransformGeminiToClaude(geminiBody, originalModel)
	if err != nil {
		logger.LegacyPrintf("service.antigravity_gateway", "[antigravity-Forward] transform_error error=%v body=%s", err, string(geminiBody))
		return nil, s.writeClaudeErrorContext(c, http.StatusBadGateway, "upstream_error", "Failed to parse upstream response")
	}

	c.SetHeader("Content-Type", "application/json")
	_, _ = c.WriteBytes(http.StatusOK, claudeResp)

	// 转换为 service.ClaudeUsage
	usage := &ClaudeUsage{
		InputTokens:              agUsage.InputTokens,
		OutputTokens:             agUsage.OutputTokens,
		CacheCreationInputTokens: agUsage.CacheCreationInputTokens,
		CacheReadInputTokens:     agUsage.CacheReadInputTokens,
	}

	return &antigravityStreamResult{usage: usage, firstTokenMs: firstTokenMs}, nil
}

// handleClaudeStreamingResponse 处理 Claude 流式响应（Gemini SSE → Claude SSE 转换）
