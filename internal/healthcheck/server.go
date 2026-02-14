package healthcheck

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultListen = "0.0.0.0:8787"
)

func NormalizeListen(listen string) string {
	return strings.TrimSpace(listen)
}

func RegisterHealthEndpoint(mux *http.ServeMux, mode string) {
	if mux == nil {
		return
	}
	mode = strings.TrimSpace(mode)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead:
		default:
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload := map[string]any{
			"ok":   true,
			"time": time.Now().Format(time.RFC3339Nano),
		}
		if mode != "" {
			payload["mode"] = mode
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
}

func StartServer(ctx context.Context, logger *slog.Logger, listen string, mode string) (*http.Server, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		logger = slog.Default()
	}
	listen = NormalizeListen(listen)
	if listen == "" {
		return nil, errors.New("empty health listen address")
	}

	mux := http.NewServeMux()
	RegisterHealthEndpoint(mux, mode)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte("ok\n"))
	})

	ln, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}

	srv := &http.Server{
		Addr:              listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(shutdownCtx)
		cancel()
	}()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("health_server_error", "addr", listen, "error", err.Error())
		}
	}()

	logger.Info("health_server_start", "addr", listen, "health_path", "/health", "mode", strings.TrimSpace(mode))
	return srv, nil
}
