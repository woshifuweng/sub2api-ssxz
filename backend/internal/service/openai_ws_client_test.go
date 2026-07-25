package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestCoderOpenAIWSClientDialer_ProxyHTTPClientReuse(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	c1, err := impl.proxyHTTPClient("http://127.0.0.1:8080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	c2, err := impl.proxyHTTPClient("http://127.0.0.1:8080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	require.Same(t, c1, c2, "同一代理地址应复用同一个 HTTP 客户端")

	c3, err := impl.proxyHTTPClient("http://127.0.0.1:8081", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	require.NotSame(t, c1, c3, "不同代理地址应分离客户端")
}

func TestCoderOpenAIWSClientDialer_ProxyHTTPClientInvalidURL(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	_, err := impl.proxyHTTPClient("://bad", openAIWSDialHTTPVersionH2)
	require.Error(t, err)
}

func TestCoderOpenAIWSClientDialer_TransportMetricsSnapshot(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	_, err := impl.proxyHTTPClient("http://127.0.0.1:18080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	_, err = impl.proxyHTTPClient("http://127.0.0.1:18080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	_, err = impl.proxyHTTPClient("http://127.0.0.1:18081", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)

	snapshot := impl.SnapshotTransportMetrics()
	require.Equal(t, int64(1), snapshot.ProxyClientCacheHits)
	require.Equal(t, int64(2), snapshot.ProxyClientCacheMisses)
	require.InDelta(t, 1.0/3.0, snapshot.TransportReuseRatio, 0.0001)
}

func TestCoderOpenAIWSClientDialer_ProxyClientCacheCapacity(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	total := openAIWSProxyClientCacheMaxEntries + 32
	for i := 0; i < total; i++ {
		_, err := impl.proxyHTTPClient(fmt.Sprintf("http://127.0.0.1:%d", 20000+i), openAIWSDialHTTPVersionH2)
		require.NoError(t, err)
	}

	impl.proxyMu.Lock()
	cacheSize := len(impl.proxyClients)
	impl.proxyMu.Unlock()

	require.LessOrEqual(t, cacheSize, openAIWSProxyClientCacheMaxEntries, "代理客户端缓存应受容量上限约束")
}

func TestCoderOpenAIWSClientDialer_ProxyClientCacheIdleTTL(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	oldProxy := "http://127.0.0.1:28080"
	_, err := impl.proxyHTTPClient(oldProxy, openAIWSDialHTTPVersionH2)
	require.NoError(t, err)

	impl.proxyMu.Lock()
	oldEntry := impl.proxyClients[oldProxy+"|"+string(openAIWSDialHTTPVersionH2)]
	require.NotNil(t, oldEntry)
	oldEntry.lastUsedUnixNano = time.Now().Add(-openAIWSProxyClientCacheIdleTTL - time.Minute).UnixNano()
	impl.proxyMu.Unlock()

	// 触发一次新的代理获取，驱动 TTL 清理。
	_, err = impl.proxyHTTPClient("http://127.0.0.1:28081", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)

	impl.proxyMu.Lock()
	_, exists := impl.proxyClients[oldProxy+"|"+string(openAIWSDialHTTPVersionH2)]
	impl.proxyMu.Unlock()

	require.False(t, exists, "超过空闲 TTL 的代理客户端应被回收")
}

func TestCoderOpenAIWSClientDialer_ProxyTransportTLSHandshakeTimeout(t *testing.T) {
	dialer := newDefaultOpenAIWSClientDialer(nil)
	impl, ok := dialer.(*coderOpenAIWSClientDialer)
	require.True(t, ok)

	client, err := impl.proxyHTTPClient("http://127.0.0.1:38080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	require.NotNil(t, client)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport)
	require.Equal(t, 10*time.Second, transport.TLSHandshakeTimeout)
}

func TestBuildOpenAIWSHTTPClient_HTTPVersionSwitch(t *testing.T) {
	h1Client := buildOpenAIWSHTTPClient(nil, openAIWSDialHTTPVersionH1)
	h1Transport, ok := h1Client.Transport.(*http.Transport)
	require.True(t, ok)
	require.False(t, h1Transport.ForceAttemptHTTP2)
	require.NotNil(t, h1Transport.TLSNextProto)

	h2Client := buildOpenAIWSHTTPClient(nil, openAIWSDialHTTPVersionH2)
	h2Transport, ok := h2Client.Transport.(*http.Transport)
	require.True(t, ok)
	require.True(t, h2Transport.ForceAttemptHTTP2)
	require.Nil(t, h2Transport.TLSNextProto)
}

func TestCoderOpenAIWSClientDialer_DialHTTPVersionConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.DialHTTPVersion = "1.1"
	impl := newDefaultOpenAIWSClientDialer(cfg).(*coderOpenAIWSClientDialer)
	require.Equal(t, openAIWSDialHTTPVersionH1, impl.dialHTTPVersion())

	cfg.Gateway.OpenAIWS.DialHTTPVersion = "2"
	impl = newDefaultOpenAIWSClientDialer(cfg).(*coderOpenAIWSClientDialer)
	require.Equal(t, openAIWSDialHTTPVersionH2, impl.dialHTTPVersion())

	cfg.Gateway.OpenAIWS.DialHTTPVersion = "auto"
	impl = newDefaultOpenAIWSClientDialer(cfg).(*coderOpenAIWSClientDialer)
	require.Equal(t, openAIWSDialHTTPVersionAuto, impl.dialHTTPVersion())

	cfg.Gateway.OpenAIWS.DialHTTPVersion = "unsupported"
	impl = newDefaultOpenAIWSClientDialer(cfg).(*coderOpenAIWSClientDialer)
	require.Equal(t, openAIWSDialHTTPVersionAuto, impl.dialHTTPVersion())
}

func TestShouldRetryOpenAIWSDialWithHTTP11(t *testing.T) {
	require.True(t, shouldRetryOpenAIWSDialWithHTTP11(http.StatusUpgradeRequired, nil, nil))
	require.True(t, shouldRetryOpenAIWSDialWithHTTP11(0, http.Header{"Server": []string{"cloudflare"}}, nil))
	require.True(t, shouldRetryOpenAIWSDialWithHTTP11(0, nil, &openAIWSDialError{Err: fmt.Errorf("websocket protocol error: Handshake not finished")}))
	require.False(t, shouldRetryOpenAIWSDialWithHTTP11(http.StatusUnauthorized, nil, nil))
	require.False(t, shouldRetryOpenAIWSDialWithHTTP11(0, nil, &openAIWSDialError{Err: tls.RecordHeaderError{}}))
}

func TestCoderOpenAIWSClientDialer_ProxyCacheSeparatesHTTPVersions(t *testing.T) {
	impl := newDefaultOpenAIWSClientDialer(nil).(*coderOpenAIWSClientDialer)

	h1Client, err := impl.proxyHTTPClient("http://127.0.0.1:48080", openAIWSDialHTTPVersionH1)
	require.NoError(t, err)
	h2Client, err := impl.proxyHTTPClient("http://127.0.0.1:48080", openAIWSDialHTTPVersionH2)
	require.NoError(t, err)
	require.NotSame(t, h1Client, h2Client)

	snapshot := impl.SnapshotTransportMetrics()
	require.Equal(t, int64(2), snapshot.ProxyClientCacheMisses)
}

func TestCoderOpenAIWSClientDialer_TransportVersionMetrics(t *testing.T) {
	impl := newDefaultOpenAIWSClientDialer(nil).(*coderOpenAIWSClientDialer)
	impl.recordOpenAIWSDialHTTPVersion("h1")
	impl.recordOpenAIWSDialHTTPVersion("h2")
	impl.fallbackH1.Add(1)

	snapshot := impl.SnapshotTransportMetrics()
	require.Equal(t, int64(1), snapshot.HTTP1DialTotal)
	require.Equal(t, int64(1), snapshot.HTTP2DialTotal)
	require.Equal(t, int64(1), snapshot.FallbackToHTTP1Total)
}

func TestCoderOpenAIWSClientDialer_PreservesHandshakeResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "request-test")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid task"}`))
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.DialHTTPVersion = "1.1"
	dialer := newDefaultOpenAIWSClientDialer(cfg)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	_, status, headers, err := dialer.Dial(t.Context(), wsURL, nil, "")

	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, status)
	require.Equal(t, "request-test", headers.Get("X-Request-Id"))
	require.Equal(t, "h1", headers.Get(openAIWSHTTPVersionHeader))
	var handshakeErr *openAIWSHandshakeError
	require.True(t, errors.As(err, &handshakeErr))
	require.JSONEq(t, `{"error":"invalid task"}`, string(handshakeErr.Body))
}

func TestCoderOpenAIWSClientConn_DoesNotSupportIdlePingWithoutReader(t *testing.T) {
	require.False(t, (&coderOpenAIWSClientConn{}).SupportsIdlePingWithoutReader())
}
