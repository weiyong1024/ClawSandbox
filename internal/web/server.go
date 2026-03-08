package web

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	docker "github.com/fsouza/go-dockerclient"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

// Server is the ClawSandbox Web UI HTTP server.
type Server struct {
	docker   *docker.Client
	config   *config.Config
	events   *EventBus
	addr     string
}

// NewServer creates a new Server.
func NewServer(cli *docker.Client, cfg *config.Config, addr string) *Server {
	return &Server{
		docker: cli,
		config: cfg,
		events: NewEventBus(),
		addr:   addr,
	}
}

// loadStore loads the state from disk. Called per-request to stay in sync with CLI.
func (s *Server) loadStore() (*state.Store, error) {
	return state.Load()
}

// ListenAndServe starts the HTTP server and blocks until interrupted.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	srv := &http.Server{
		Addr:    s.addr,
		Handler: requestLogger(mux),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pidPath := writePIDFile()

	errCh := make(chan error, 1)
	go func() {
		ln, err := net.Listen("tcp", s.addr)
		if err != nil {
			errCh <- fmt.Errorf("listen %s: %w", s.addr, err)
			return
		}
		log.Printf("ClawSandbox Web UI: http://%s", ln.Addr())
		errCh <- srv.Serve(ln)
	}()

	var result error
	select {
	case err := <-errCh:
		result = err
	case <-ctx.Done():
		log.Println("Shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result = srv.Shutdown(shutdownCtx)
	}

	removePIDFile(pidPath)
	return result
}

func writePIDFile() string {
	dir, err := config.DataDir()
	if err != nil {
		return ""
	}
	path := filepath.Join(dir, "serve.pid")
	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		log.Printf("Warning: could not write PID file: %v", err)
		return ""
	}
	return path
}

func removePIDFile(path string) {
	if path != "" {
		os.Remove(path)
	}
}

// requestLogger is a simple middleware that logs each request.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
