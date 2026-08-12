package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
)

func (s *GatewayService) executeBedrockUpstreamContext(
	ctx context.Context,
	c gatewayctx.GatewayContext,
	account *Account,
	body []byte,
	modelID string,
	region string,
	stream bool,
	signer *BedrockSigner,
	apiKey string,
	proxyURL string,
) (*http.Response, error) {
	var resp *http.Response
	var err error
	retryStart := time.Now()
	for attempt := 1; attempt <= maxRetryAttempts; attempt++ {
		var upstreamReq *http.Request
		if account.IsBedrockAPIKey() {
			upstreamReq, err = s.buildUpstreamRequestBedrockAPIKey(ctx, body, modelID, region, stream, apiKey)
		} else {
			upstreamReq, err = s.buildUpstreamRequestBedrock(ctx, body, modelID, region, stream, signer)
		}
		if err != nil {
			return nil, err
		}

		upstreamReq, cancelQuickFail := withProxyQuickFailRequest(upstreamReq, proxyURL)
		resp, err = s.httpUpstream.DoWithTLS(upstreamReq, proxyURL, account.ID, account.Concurrency, s.resolveTLSFingerprintProfile(account))
		if err != nil {
			if cancelQuickFail != nil {
				cancelQuickFail()
			}
			if resp != nil && resp.Body != nil {
				_ = resp.Body.Close()
			}
			safeErr := sanitizeUpstreamErrorMessage(err.Error())
			setOpsUpstreamErrorContext(c, 0, safeErr, "")
			appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: 0,
				Kind:               "request_error",
				Message:            safeErr,
			})
			return nil, newProxyRequestFailoverError(account, proxyURL, err)
		}
		resp = attachProxyQuickFailCancel(resp, cancelQuickFail)

		if resp.StatusCode >= 400 && resp.StatusCode != 400 && s.shouldRetryUpstreamError(account, resp.StatusCode) {
			if attempt < maxRetryAttempts {
				elapsed := time.Since(retryStart)
				if elapsed >= maxRetryElapsed {
					break
				}

				delay := retryBackoffDelay(attempt)
				remaining := maxRetryElapsed - elapsed
				if delay > remaining {
					delay = remaining
				}
				if delay <= 0 {
					break
				}

				respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
				_ = resp.Body.Close()
				appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
					Platform:           account.Platform,
					AccountID:          account.ID,
					AccountName:        account.Name,
					UpstreamStatusCode: resp.StatusCode,
					Kind:               "retry",
					Message:            extractUpstreamErrorMessage(respBody),
					Detail: func() string {
						if s.cfg != nil && s.cfg.Gateway.LogUpstreamErrorBody {
							return truncateString(string(respBody), s.cfg.Gateway.LogUpstreamErrorBodyMaxBytes)
						}
						return ""
					}(),
				})
				logger.LegacyPrintf("service.gateway", "[Bedrock] account %d: upstream error %d, retry %d/%d after %v",
					account.ID, resp.StatusCode, attempt, maxRetryAttempts, delay)
				if err := sleepWithContext(ctx, delay); err != nil {
					return nil, err
				}
				continue
			}
			break
		}

		break
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("upstream request failed: empty response")
	}
	return resp, nil
}

// handleBedrockUpstreamErrors 处理 Bedrock 上游 4xx/5xx 错误（failover + 错误响应）

func (s *GatewayService) handleBedrockUpstreamErrorsContext(
	ctx context.Context,
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
) (*ForwardResult, error) {
	// retry exhausted + failover
	if s.shouldRetryUpstreamError(account, resp.StatusCode) {
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			_ = resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewReader(respBody))

			logger.LegacyPrintf("service.gateway", "[Bedrock] Upstream error (retry exhausted, failover): Account=%d(%s) Status=%d Body=%s",
				account.ID, account.Name, resp.StatusCode, truncateString(string(respBody), 1000))

			s.handleRetryExhaustedSideEffects(ctx, resp, account)
			appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
				Platform:           account.Platform,
				AccountID:          account.ID,
				AccountName:        account.Name,
				UpstreamStatusCode: resp.StatusCode,
				Kind:               "retry_exhausted_failover",
				Message:            extractUpstreamErrorMessage(respBody),
			})
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return s.handleRetryExhaustedErrorContext(ctx, resp, c, account)
	}

	// non-retryable failover
	if s.shouldFailoverUpstreamError(resp.StatusCode) {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))

		s.handleFailoverSideEffects(ctx, resp, account)
		appendOpsUpstreamErrorContext(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			Kind:               "failover",
			Message:            extractUpstreamErrorMessage(respBody),
		})
		return nil, &UpstreamFailoverError{
			StatusCode:             resp.StatusCode,
			ResponseBody:           respBody,
			RetryableOnSameAccount: account.IsPoolMode() && isPoolModeRetryableStatus(resp.StatusCode),
		}
	}

	// other errors
	return s.handleErrorResponseContext(ctx, resp, c, account)
}

// buildUpstreamRequestBedrock 构建 Bedrock 上游请求

func (s *GatewayService) handleBedrockNonStreamingResponseContext(
	ctx context.Context,
	resp *http.Response,
	c gatewayctx.GatewayContext,
	account *Account,
) (*ClaudeUsage, error) {
	maxBytes := resolveUpstreamResponseReadLimit(s.cfg)
	body, err := readUpstreamResponseBodyLimited(resp.Body, maxBytes)
	if err != nil {
		if errors.Is(err, ErrUpstreamResponseBodyTooLarge) {
			setOpsUpstreamErrorContext(c, http.StatusBadGateway, "upstream response too large", "")
			c.WriteJSON(http.StatusBadGateway, gin.H{
				"type": "error",
				"error": gin.H{
					"type":    "upstream_error",
					"message": "Upstream response too large",
				},
			})
		}
		return nil, err
	}

	// 转换 Bedrock 特有的 amazon-bedrock-invocationMetrics 为标准 Anthropic usage 格式
	// 并移除该字段避免透传给客户端
	body = transformBedrockInvocationMetrics(body)

	usage := parseClaudeUsageFromResponseBody(body)

	c.SetHeader("Content-Type", "application/json")
	if v := resp.Header.Get("x-amzn-requestid"); v != "" {
		c.SetHeader("x-request-id", v)
	}
	_, _ = c.WriteBytes(resp.StatusCode, body)
	return usage, nil
}
