package metrics

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	registry := NewRegistry()

	server, err := NewServer(ServerConfig{
		Addr:     ":9200",
		Gatherer: registry.Gatherer(),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}
	if server.httpServer.Addr != ":9200" {
		t.Errorf("http server address = %q, want %q", server.httpServer.Addr, ":9200")
	}
	if server.httpServer.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", server.httpServer.ReadHeaderTimeout, 5*time.Second)
	}
	if server.httpServer.Handler == nil {
		t.Fatal("http server handler is nil")
	}
}

func TestNewServerRejectsInvalidConfig(t *testing.T) {
	registry := NewRegistry()
	tests := []struct {
		name   string
		config ServerConfig
	}{
		{
			name: "empty address",
			config: ServerConfig{
				Gatherer: registry.Gatherer(),
			},
		},
		{
			name: "whitespace address",
			config: ServerConfig{
				Addr:     "   ",
				Gatherer: registry.Gatherer(),
			},
		},
		{
			name: "nil gatherer",
			config: ServerConfig{
				Addr: ":9200",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, err := NewServer(test.config)
			if !errors.Is(err, ErrInvalidConfig) {
				t.Fatalf("NewServer() error = %v, want %v", err, ErrInvalidConfig)
			}
			if server != nil {
				t.Errorf("NewServer() server = %v, want nil", server)
			}
		})
	}
}

func TestServerMetricsRoute(t *testing.T) {
	registry := NewRegistry()
	server, err := NewServer(ServerConfig{
		Addr:     ":9200",
		Gatherer: registry.Gatherer(),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "go_goroutines") {
		t.Error("GET /metrics response does not contain go_goroutines")
	}
}

func TestServerDoesNotExposeOtherRoutes(t *testing.T) {
	registry := NewRegistry()
	server, err := NewServer(ServerConfig{
		Addr:     ":9200",
		Gatherer: registry.Gatherer(),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Errorf("GET / status = %d, want %d", response.Code, http.StatusNotFound)
	}
}

func TestServerServeAndShutdown(t *testing.T) {
	addr := availableTCPAddr(t)
	registry := NewRegistry()
	server, err := NewServer(ServerConfig{
		Addr:     addr,
		Gatherer: registry.Gatherer(),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve()
	}()

	waitForMetricsServer(t, "http://"+addr+"/metrics")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	select {
	case err := <-serveErr:
		if err != nil {
			t.Errorf("Serve() after Shutdown() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve() did not return after Shutdown()")
	}
}

func TestServerServeReturnsListenError(t *testing.T) {
	registry := NewRegistry()
	server, err := NewServer(ServerConfig{
		Addr:     "127.0.0.1:not-a-port",
		Gatherer: registry.Gatherer(),
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	err = server.Serve()
	if err == nil {
		t.Fatal("Serve() error = nil, want listen error")
	}
	if errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("Serve() error = %v, want non-shutdown error", err)
	}
}

func availableTCPAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for available address: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close address reservation: %v", err)
	}
	return addr
}

func waitForMetricsServer(t *testing.T, url string) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			_, copyErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if copyErr != nil {
				t.Fatalf("read metrics response: %v", copyErr)
			}
			if closeErr != nil {
				t.Fatalf("close metrics response: %v", closeErr)
			}
			if response.StatusCode != http.StatusOK {
				t.Fatalf("GET /metrics status = %d, want %d", response.StatusCode, http.StatusOK)
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("metrics server at %s did not become ready", url)
}
