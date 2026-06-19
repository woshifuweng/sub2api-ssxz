package service

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/gatewayctx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_JSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-1","prompt":"draw a cat","size":"1024x1024","quality":"high","stream":true}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "/v1/images/generations", parsed.Endpoint)
	require.Equal(t, "gpt-image-1", parsed.Model)
	require.Equal(t, "draw a cat", parsed.Prompt)
	require.True(t, parsed.Stream)
	require.Equal(t, "1024x1024", parsed.Size)
	require.Equal(t, "1K", parsed.SizeTier)
	require.Equal(t, OpenAIImagesCapabilityNative, parsed.RequiredCapability)
	require.False(t, parsed.Multipart)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_MultipartEdit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "gpt-image-1"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	part, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = part.Write([]byte("fake-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(body.Bytes()))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body.Bytes())
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "/v1/images/edits", parsed.Endpoint)
	require.True(t, parsed.Multipart)
	require.Equal(t, "gpt-image-1", parsed.Model)
	require.Equal(t, "replace background", parsed.Prompt)
	require.Equal(t, "2K", parsed.SizeTier)
	require.Len(t, parsed.Uploads, 1)
	require.Equal(t, OpenAIImagesCapabilityChatWebEdit, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_PromptOnlyDefaultsRemainBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"prompt":"draw a cat"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "gpt-image-1", parsed.Model)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_ExplicitSizeRequiresNativeCapability(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"prompt":"draw a cat","size":"1024x1024"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, OpenAIImagesCapabilityNative, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceParseOpenAIImagesRequest_ExplicitImageModelStillBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat","n":2,"response_format":"url"}`)

	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = req

	svc := &OpenAIGatewayService{}
	parsed, err := svc.ParseOpenAIImagesRequest(c, body)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	require.Equal(t, "gpt-image-2", parsed.Model)
	require.Equal(t, 2, parsed.N)
	require.Equal(t, "url", parsed.ResponseFormat)
	require.Equal(t, OpenAIImagesCapabilityBasic, parsed.RequiredCapability)
}

func TestOpenAIGatewayServiceForwardImagesContext_Upstream4xxReturnsErrorResult(t *testing.T) {
	svc, ctx, rec, body, parsed, account := newOpenAIImagesForwardTestHarness(t, &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newJSONResponse(http.StatusBadRequest, `{"error":{"message":"invalid image request"}}`),
		},
	})

	result, err := svc.ForwardImagesContext(ctx.Request.Context(), gatewayctx.FromGin(ctx), account, body, parsed, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.Equal(t, http.StatusBadGateway, ctx.Writer.Status())
	require.Contains(t, rec.Body.String(), "Upstream request failed")
}

func TestOpenAIGatewayServiceForwardImagesContext_Upstream5xxReturnsFailoverErrorResult(t *testing.T) {
	svc, ctx, rec, body, parsed, account := newOpenAIImagesForwardTestHarness(t, &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			newJSONResponse(http.StatusInternalServerError, `{"error":{"message":"upstream unavailable"}}`),
		},
	})

	result, err := svc.ForwardImagesContext(ctx.Request.Context(), gatewayctx.FromGin(ctx), account, body, parsed, "")

	require.Error(t, err)
	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusInternalServerError, failoverErr.StatusCode)
	require.False(t, ctx.Writer.Written())
	require.Empty(t, rec.Body.String())
}

func TestOpenAIGatewayServiceForwardImagesContext_TransportErrorReturnsErrorResult(t *testing.T) {
	svc, ctx, _, body, parsed, account := newOpenAIImagesForwardTestHarness(t, &queuedHTTPUpstreamStub{
		errors: []error{contextDeadlineExceededForImagesTest{}},
	})

	result, err := svc.ForwardImagesContext(ctx.Request.Context(), gatewayctx.FromGin(ctx), account, body, parsed, "")

	require.Error(t, err)
	require.Nil(t, result)
	require.False(t, ctx.Writer.Written())
}

func TestOpenAIGatewayServiceForwardImagesContext_PartialSuccessWriteFailureReturnsErrorResult(t *testing.T) {
	ctx, _ := newOpenAIImagesGatewayContext(t, `{"model":"gpt-image-2","prompt":"draw a cat"}`)
	failingCtx := &failingWriteGatewayContext{
		GatewayContext: gatewayctx.FromGin(ctx),
	}
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)
	parsed, err := (&OpenAIGatewayService{}).ParseOpenAIImagesRequestContext(failingCtx, body)
	require.NoError(t, err)
	svc := &OpenAIGatewayService{
		httpUpstream: &queuedHTTPUpstreamStub{
			responses: []*http.Response{
				newJSONResponse(http.StatusOK, `{"created":1740000000,"data":[{"url":"https://cdn.example.com/work.png"}]}`),
			},
		},
	}
	account := openAIImagesForwardTestAccount()

	result, err := svc.ForwardImagesContext(failingCtx.Request().Context(), failingCtx, account, body, parsed, "")

	require.Error(t, err)
	require.Nil(t, result)
}

type contextDeadlineExceededForImagesTest struct{}

func (contextDeadlineExceededForImagesTest) Error() string   { return "context deadline exceeded" }
func (contextDeadlineExceededForImagesTest) Timeout() bool   { return true }
func (contextDeadlineExceededForImagesTest) Temporary() bool { return true }

type failingWriteGatewayContext struct {
	gatewayctx.GatewayContext
}

func (c *failingWriteGatewayContext) WriteBytes(status int, payload []byte) (int, error) {
	return 0, errors.New("client disconnected")
}

func newOpenAIImagesForwardTestHarness(t *testing.T, upstream *queuedHTTPUpstreamStub) (*OpenAIGatewayService, *gin.Context, *httptest.ResponseRecorder, []byte, *OpenAIImagesRequest, *Account) {
	t.Helper()
	body := []byte(`{"model":"gpt-image-2","prompt":"draw a cat"}`)
	ctx, rec := newOpenAIImagesGatewayContext(t, string(body))
	svc := &OpenAIGatewayService{httpUpstream: upstream}
	parsed, err := svc.ParseOpenAIImagesRequestContext(gatewayctx.FromGin(ctx), body)
	require.NoError(t, err)
	return svc, ctx, rec, body, parsed, openAIImagesForwardTestAccount()
}

func newOpenAIImagesGatewayContext(t *testing.T, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	return ctx, rec
}

func openAIImagesForwardTestAccount() *Account {
	return &Account{
		ID:          9001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "https://api.example.test",
		},
		CreatedAt: time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
	}
}
