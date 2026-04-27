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

	"github.com/clawfleet/clawfleet/internal/config"
	"github.com/clawfleet/clawfleet/internal/state"
)

// Server is the ClawFleet Web UI HTTP server.
type Server struct {
	docker     *docker.Client
	config     *config.Config
	events     *EventBus
	addr       string
	codexFlows *codexFlowManager
}

// NewServer creates a new Server.
func NewServer(cli *docker.Client, cfg *config.Config, addr string) *Server {
	s := &Server{
		docker:     cli,
		config:     cfg,
		events:     NewEventBus(),
		addr:       addr,
		codexFlows: newCodexFlowManager(),
	}
	s.codexFlows.startCallbackRelay()
	return s
}

// loadStore loads the state from disk. Called per-request to stay in sync with CLI.
func (s *Server) loadStore() (*state.Store, error) {
	return state.Load()
}

// ListenAndServe starts the HTTP server and blocks until interrupted.
func (s *Server) ListenAndServe() error {
	s.runMigrations()

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	srv := &http.Server{
		Addr:    s.addr,
		Handler: requestLogger(mux),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := s.checkExistingDashboard(); err != nil {
		return err
	}

	pidPath := writePIDFile()

	errCh := make(chan error, 1)
	go func() {
		ln, err := net.Listen("tcp", s.addr)
		if err != nil {
			errCh <- fmt.Errorf("listen %s: %w", s.addr, err)
			return
		}
		log.Printf("ClawFleet Web UI: http://%s", ln.Addr())
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

// checkExistingDashboard checks if another dashboard is already running,
// first via PID file, then by probing the listen address.
func (s *Server) checkExistingDashboard() error {
	// Check PID file first.
	dir, err := config.DataDir()
	if err == nil {
		pidPath := filepath.Join(dir, "serve.pid")
		if data, err := os.ReadFile(pidPath); err == nil {
			if pid, err := strconv.Atoi(string(data)); err == nil {
				if proc, err := os.FindProcess(pid); err == nil {
					if err := proc.Signal(syscall.Signal(0)); err == nil {
						return fmt.Errorf("Dashboard is already running (pid %d).\n"+
							"Run 'clawfleet dashboard stop' first, or use '--port' to listen on a different port", pid)
					}
				}
			}
		}
	}

	// Fallback: probe the port directly.
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("port %s is already in use.\n"+
			"Stop the process using that port, or use '--port' to listen on a different port", s.addr)
	}
	ln.Close()
	return nil
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
