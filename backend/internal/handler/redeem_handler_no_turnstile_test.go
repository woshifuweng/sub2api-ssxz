package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// stubRedeemRepoNotFound 只实现 GetByCode，其余方法由嵌入的接口占位（值为 nil）。
// 这样做是故意的：如果执行路径变化导致调用了别的方法，会立刻 nil panic 而不是静默通过。
type stubRedeemRepoNotFound struct {
	service.RedeemCodeRepository
	called int
}

func (s *stubRedeemRepoNotFound) GetByCode(ctx context.Context, code string) (*service.RedeemCode, error) {
	s.called++
	return nil, service.ErrRedeemCodeNotFound
}

// TestRedeemGatewayDoesNotRequireTurnstile 锁定「兑换码不做人机验证」这个决定。
//
// 背景：2026-07-18 给兑换码加了 Turnstile 前置校验，之后 widget 卡在"正在验证…"
// 会让兑换完全不可用（客户拿着卡密兑不进来）。这个决定此前口头定过一次但没有落成检查，
// 结果又被加回来了——所以这里用一条会失败的测试把它钉住。
//
// 断言方式：故意把 Turnstile.Required 设为 true。即便如此，请求也必须一路抵达
// RedeemService（repo.called == 1）并按兑换码不存在返回 404。
// 如果有人重新加上验证码前置闸，请求会在抵达 service 之前就返回，
// repo.called 变成 0，本测试失败。
//
// 注意：登录/注册/找回密码/支付仍保留 Turnstile，不在本测试范围内。
func TestRedeemGatewayDoesNotRequireTurnstile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &stubRedeemRepoNotFound{}
	h := &RedeemHandler{
		// cache 传 nil：RedeemService 的限流/加锁辅助函数都有 nil 保护，
		// 会降级放行，因此不需要为它造假实现。
		redeemService: service.NewRedeemService(repo, nil, nil, nil, nil, nil, nil),
		authService: service.NewAuthService(nil, nil, nil, nil, &config.Config{
			Server:    config.ServerConfig{Mode: "release"},
			Turnstile: config.TurnstileConfig{Required: true},
		}, nil, nil, nil, nil, nil, nil),
	}

	router := gin.New()
	router.Use(withUserSubject(42))
	router.POST("/api/v1/redeem", func(c *gin.Context) {
		h.RedeemGateway(gatewayctx.FromGin(c))
	})

	// 前端现在不再发 turnstile_token，这里发的就是前端的真实载荷。
	req := httptest.NewRequest(http.MethodPost, "/api/v1/redeem", bytes.NewBufferString(`{"code":"TEST-CODE"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, 1, repo.called,
		"请求必须抵达 RedeemService；为 0 说明兑换码又被加上了人机验证前置闸")
	require.Equal(t, http.StatusNotFound, rec.Code,
		"应按「兑换码不存在」返回 404，而不是验证码相关错误")
}
