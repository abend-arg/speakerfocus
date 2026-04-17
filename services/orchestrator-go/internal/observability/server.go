package observability

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsServer struct {
	addr   string
	path   string
	server *http.Server
}

func StartMetricsServer(ctx context.Context, addr string, path string, registry *prometheus.Registry) (*MetricsServer, error) {
	if addr == "" {
		return nil, nil
	}
	if path == "" {
		path = "/metrics"
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen metrics server: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	metricsServer := &MetricsServer{
		addr:   listener.Addr().String(),
		path:   path,
		server: server,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("metrics server failed: %v\n", err)
		}
	}()

	return metricsServer, nil
}

func (s *MetricsServer) Addr() string {
	if s == nil {
		return ""
	}
	return s.addr
}

func (s *MetricsServer) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}
