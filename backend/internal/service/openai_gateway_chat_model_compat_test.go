package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardAsChatCompletionsContext_ChatOnlyAPIKeyKeepsChatProtocol(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-5.4-mini","messages":[{"role":"user","content":"hello"}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"rid-chat-context"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"chatcmpl_context","object":"chat.completion","model":"gpt-5.4-mini","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}}`,
		)),
	}}
	svc := &OpenAIGatewayService{cfg: chatCompatRegressionConfig(), httpUpstream: upstream}
	account := chatCompatRegressionAccount()
	account.Extra = map[string]any{
		openai_compat.ExtraKeyResponsesSupported: false,
	}

	result, err := svc.ForwardAsChatCompletionsContext(
		context.Background(),
		gatewayctx.FromGin(c),
		account,
		body,
		"",
		"",
	)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://upstream.example/v1/chat/completions", upstream.lastReq.URL.String())
	require.Equal(t, "hello", gjson.GetBytes(upstream.lastBody, "messages.0.content").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "input").Exists())
	require.Equal(t, "chat.completion", gjson.Get(recorder.Body.String(), "object").String())
}

func TestForwardResponsesContext_AutoSupportedPassthroughPreservesStructuredInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	body := []byte(`{"model":"gpt-5.4-mini","input":[{"role":"user","content":[{"type":"input_text","text":"enterprise regression"}]}],"stream":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "X-Request-Id": []string{"rid-enterprise-responses"}},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"resp_enterprise","object":"response","model":"gpt-5.4-mini","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"ok"}],"status":"completed"}],"usage":{"input_tokens":5,"output_tokens":2,"total_tokens":7}}`,
		)),
	}}
	svc := &OpenAIGatewayService{cfg: chatCompatRegressionConfig(), httpUpstream: upstream}
	account := chatCompatRegressionAccount()
	account.Extra = map[string]any{
		"openai_passthrough":                     true,
		openai_compat.ExtraKeyResponsesSupported: true,
	}

	result, err := svc.ForwardContext(context.Background(), gatewayctx.FromGin(c), account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "http://upstream.example/v1/responses", upstream.lastReq.URL.String())
	require.Equal(t, "enterprise regression", gjson.GetBytes(upstream.lastBody, "input.0.content.0.text").String())
	require.False(t, gjson.GetBytes(upstream.lastBody, "messages").Exists())
	require.Equal(t, "ok", gjson.Get(recorder.Body.String(), "output.0.content.0.text").String())
}

func chatCompatRegressionConfig() *config.Config {
	return &config.Config{
		Security: config.SecurityConfig{
			URLAllowlist: config.URLAllowlistConfig{
				Enabled:           false,
				AllowInsecureHTTP: true,
			},
		},
	}
}

func chatCompatRegressionAccount() *Account {
	return &Account{
		ID:          901,
		Name:        "chat-compat-regression",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "http://upstream.example",
		},
	}
}
