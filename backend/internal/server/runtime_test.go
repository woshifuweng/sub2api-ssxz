package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestBuildRouteManifestIncludesHintsAndSorts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	router.GET("/responses", func(c *gin.Context) {
		c.Status(http.StatusSwitchingProtocols)
	})

	manifest := BuildRouteManifest(router)
	requireSortedRouteManifest(t, manifest)

	health := requireRouteManifestEntry(t, manifest, "GET", "/health")
	require.NotEmpty(t, health.Handler)
	require.True(t, health.Executable)
	require.Contains(t, health.Middleware, "request_logger")

	responses := requireRouteManifestEntry(t, manifest, "GET", "/responses")
	require.True(t, responses.Hints.Streaming)
	require.True(t, responses.Hints.WebSocket)
}

func TestProvideHTTPServerRegistersRouteManifestMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              8080,
			ReadHeaderTimeout: 5,
			IdleTimeout:       30,
			RuntimeMode:       config.ServerRuntimeModeNetHTTP,
		},
	}
	router := gin.New()
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	httpServer := ProvideHTTPServer(cfg, router)
	runtime := ResolveIngressRuntime(cfg, httpServer)

	require.Equal(t, config.ServerRuntimeModeNetHTTP, runtime.Name())
	require.Equal(t, cfg.Server.Address(), runtime.Addr())
	manifest := runtime.RouteManifest()
	requireSortedRouteManifest(t, manifest)
	requireRouteManifestEntry(t, manifest, "GET", "/health")
}

func TestResolveIngressRuntimeHonorsConfiguredMode(t *testing.T) {
	server := &http.Server{Addr: "127.0.0.1:8080"}

	cfg := &config.Config{}
	cfg.Server.RuntimeMode = config.ServerRuntimeModeHybrid
	require.Equal(t, config.ServerRuntimeModeHybrid, ResolveIngressRuntime(cfg, server).Name())

	cfg.Server.RuntimeMode = config.ServerRuntimeModeGnet
	require.Equal(t, config.ServerRuntimeModeGnet, ResolveIngressRuntime(cfg, server).Name())
}

func requireSortedRouteManifest(t *testing.T, manifest RouteManifest) {
	t.Helper()

	for i := 1; i < len(manifest); i++ {
		prev := manifest[i-1]
		curr := manifest[i]
		if prev.Path == curr.Path {
			require.LessOrEqual(t, prev.Method, curr.Method)
			continue
		}
		require.LessOrEqual(t, prev.Path, curr.Path)
	}
}

func requireRouteManifestEntry(t *testing.T, manifest RouteManifest, method, path string) RouteManifestEntry {
	t.Helper()

	for _, entry := range manifest {
		if entry.Method == method && entry.Path == path {
			return entry
		}
	}
	require.Failf(t, "missing route manifest entry", "%s %s not found in %#v", method, path, manifest)
	return RouteManifestEntry{}
}

func TestNewRustSidecarIngressRuntimeWhenEnabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = "/tmp/sub2api-rust-sidecar.sock"
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)
	require.Equal(t, "rust-sidecar-h2c", rt.Name())
	require.Equal(t, "127.0.0.1:8080", rt.Addr())
}

func TestRustSidecarIngressRuntimeHealthcheck(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer ln.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"test-sidecar","version":"v0"}`))
	})
	srv := &http.Server{Handler: mux}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()
	go func() {
		_ = srv.Serve(ln)
	}()

	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = socketPath
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)
	require.NoError(t, rt.healthcheck())
}

func TestRustSidecarIngressRuntimeProxiesRawConnectionToUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket proxying is exercised on Unix-like platforms")
	}

	socketPath := filepath.Join(t.TempDir(), "rust-sidecar.sock")
	ln, err := net.Listen("unix", socketPath)
	require.NoError(t, err)
	defer ln.Close()

	sidecarDone := make(chan struct{})
	go func() {
		defer close(sidecarDone)
		healthBody := `{"status":"ok","service":"test-sidecar","version":"v0"}`
		for i := 0; i < 2; i++ {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				peek, err := reader.Peek(4)
				if err == nil && string(peek) == "GET " {
					for {
						line, readErr := reader.ReadString('\n')
						if readErr != nil || line == "\r\n" {
							break
						}
					}
					_, _ = c.Write([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s", len(healthBody), healthBody)))
					return
				}
				buf := make([]byte, 4)
				_, _ = io.ReadFull(reader, buf)
				_, _ = c.Write([]byte("pong"))
			}(conn)
		}
	}()

	cfg := &config.Config{}
	cfg.Rust.Sidecar.Enabled = true
	cfg.Rust.Sidecar.SocketPath = socketPath
	cfg.Rust.Sidecar.FailClosed = true
	cfg.Rust.Sidecar.RequestTimeoutSeconds = 1
	cfg.Rust.Sidecar.HealthcheckTimeoutSeconds = 1

	listener := newClassifiedListener(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	rt := newRustSidecarIngressRuntime(cfg, &http.Server{Addr: "127.0.0.1:8080"}, nil)
	require.NotNil(t, rt)

	errCh := make(chan error, 1)
	go func() {
		errCh <- rt.Serve(listener)
	}()

	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	require.True(t, listener.enqueue(serverConn))

	_, err = clientConn.Write([]byte("ping"))
	require.NoError(t, err)
	buf := make([]byte, 4)
	_, err = io.ReadFull(clientConn, buf)
	require.NoError(t, err)
	require.Equal(t, "pong", string(buf))

	require.NoError(t, listener.Close())
	select {
	case serveErr := <-errCh:
		require.NoError(t, serveErr)
	case <-time.After(2 * time.Second):
		t.Fatal("rust sidecar runtime did not exit in time")
	}

	select {
	case <-sidecarDone:
	case <-time.After(2 * time.Second):
		t.Fatal("sidecar stub did not exit in time")
	}
}

func TestClassifyPreface(t *testing.T) {
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte(http2.ClientPreface)))
	require.Equal(t, protocolTargetH2C, classifyPreface([]byte("GET / HTTP/1.1\r\nHost: example.com\r\nUpgrade: h2c\r\nHTTP2-Settings: AAMAAABkAAQCAAAAAAIAAAAA\r\n\r\n")))
	require.Equal(t, protocolTargetHTTP1, classifyPreface([]byte("GET /health HTTP/1.1\r\nHost: example.com\r\n\r\n")))
}

func TestClassifyPrefaceRoutesResponsesWebSocketToSidecarWhenEnabled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, true, false, true)
	target := classifyPrefaceWithOptions([]byte("GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetSidecar, target)
}

func TestClassifyPrefaceDoesNotRouteResponsesWebSocketToSidecarWhenDisabled(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, false, false, false)
	target := classifyPrefaceWithOptions([]byte("GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetHTTP1, target)
}

func TestClassifyPrefaceMatchesMixedCaseHeadersWithoutStringLowerAlloc(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	mux := newProtocolMux(ln, 0, true, false, true)
	target := classifyPrefaceWithOptions([]byte("GeT /V1/Responses HTTP/1.1\r\nHost: example.com\r\nUpGrAdE: WebSocket\r\nConnection: Upgrade\r\n\r\n"), mux)
	require.Equal(t, protocolTargetSidecar, target)
}

func TestHasHTTPHeaderTerminatorIncrementalDetectsBoundaryAcrossReadChunks(t *testing.T) {
	buf := []byte("GET /health HTTP/1.1\r\nHost: example.com\r\n")
	require.False(t, hasHTTPHeaderTerminatorIncremental(buf, 0))

	buf = append(buf, '\r', '\n')
	require.True(t, hasHTTPHeaderTerminatorIncremental(buf, len(buf)-2))
}

func TestClassifiedListenerBuffersBurstWithoutConcurrentAccept(t *testing.T) {
	listener := newClassifiedListener(&net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	serverConn, clientConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()
	require.True(t, listener.enqueue(serverConn))
}

func TestHybridRuntimeRoutesResponsesWebSocketUpgradeToSidecar(t *testing.T) {
	preface := []byte("GET /v1/responses HTTP/1.1\r\nHost: example.com\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n")

	requireProtocolMuxRoutesPrefaceToSidecar(t, preface, false, true)
}

func TestHybridRuntimeRoutesH2CToSidecar(t *testing.T) {
	requireProtocolMuxRoutesPrefaceToSidecar(t, []byte(http2.ClientPreface), true, false)
}

func requireProtocolMuxRoutesPrefaceToSidecar(t *testing.T, preface []byte, routeH2CToSidecar, routeResponsesWSToSidecar bool) {
	t.Helper()

	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	mux := newProtocolMux(base, time.Second, true, routeH2CToSidecar, routeResponsesWSToSidecar)
	defer func() {
		require.NoError(t, mux.Close())
	}()

	serverConn, clientConn := net.Pipe()
	defer func() {
		require.NoError(t, clientConn.Close())
	}()

	done := make(chan struct{})
	go func() {
		defer close(done)
		mux.dispatch(serverConn)
	}()

	_, err = clientConn.Write(preface)
	require.NoError(t, err)

	sidecarConn, err := acceptClassifiedConn(mux.SidecarListener(), time.Second)
	require.NoError(t, err)

	got := make([]byte, len(preface))
	_, err = io.ReadFull(sidecarConn, got)
	require.NoError(t, err)
	require.Equal(t, preface, got)
	require.NoError(t, sidecarConn.Close())

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("protocol mux dispatch did not finish")
	}
}

func acceptClassifiedConn(listener net.Listener, timeout time.Duration) (net.Conn, error) {
	type acceptResult struct {
		conn net.Conn
		err  error
	}
	ch := make(chan acceptResult, 1)
	go func() {
		conn, err := listener.Accept()
		ch <- acceptResult{conn: conn, err: err}
	}()
	select {
	case result := <-ch:
		return result.conn, result.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for classified connection")
	}
}
