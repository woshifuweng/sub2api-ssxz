package service

import (
	"context"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// HTTPUpstream 上游 HTTP 请求接口
// 用于向上游 API（Claude、OpenAI、Gemini 等）发送请求
type HTTPUpstream interface {
	// Do 执行 HTTP 请求（不启用 TLS 指纹）
	Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error)

	// DoWithTLS 执行带 TLS 指纹伪装的 HTTP 请求
	//
	// profile 参数:
	//   - nil: 不启用 TLS 指纹，行为与 Do 方法相同
	//   - non-nil: 使用指定的 Profile 进行 TLS 指纹伪装
	//
	// Profile 由调用方通过 TLSFingerprintProfileService 解析后传入，
	// 支持按账号绑定的数据库 profile 或内置默认 profile。
	DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error)
}

type upstreamTransportOverrideContextKey struct{}

// UpstreamTransportOverride carries request-scoped transport timeout overrides.
type UpstreamTransportOverride struct {
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
}

func WithUpstreamTransportOverride(ctx context.Context, override UpstreamTransportOverride) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if override.DialTimeout <= 0 && override.ResponseHeaderTimeout <= 0 {
		return ctx
	}
	return context.WithValue(ctx, upstreamTransportOverrideContextKey{}, override)
}

func GetUpstreamTransportOverride(ctx context.Context) (UpstreamTransportOverride, bool) {
	if ctx == nil {
		return UpstreamTransportOverride{}, false
	}
	override, ok := ctx.Value(upstreamTransportOverrideContextKey{}).(UpstreamTransportOverride)
	if !ok || (override.DialTimeout <= 0 && override.ResponseHeaderTimeout <= 0) {
		return UpstreamTransportOverride{}, false
	}
	return override, true
}
