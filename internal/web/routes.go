package web

import (
	"io/fs"
	"net/http"
)

// registerRoutes wires all API and static file routes.
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// REST API
	mux.HandleFunc("GET /api/v1/instances", s.handleListInstances)
	mux.HandleFunc("POST /api/v1/instances", s.handleCreateInstances)
	mux.HandleFunc("POST /api/v1/instances/{name}/start", s.handleStartInstance)
	mux.HandleFunc("POST /api/v1/instances/{name}/stop", s.handleStopInstance)
	mux.HandleFunc("DELETE /api/v1/instances/{name}", s.handleDestroyInstance)
	mux.HandleFunc("GET /api/v1/instances/{name}/logs", s.handleInstanceLogs)
	mux.HandleFunc("GET /api/v1/image/status", s.handleImageStatus)
	mux.HandleFunc("POST /api/v1/image/build", s.handleImageBuild)

	// WebSocket endpoints
	mux.HandleFunc("GET /api/v1/ws/stats", s.handleWSStats)
	mux.HandleFunc("GET /api/v1/ws/logs/{name}", s.handleWSLogs)
	mux.HandleFunc("GET /api/v1/ws/events", s.handleWSEvents)

	// Static files (frontend)
	staticSub, _ := fs.Sub(StaticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticSub)))
}
