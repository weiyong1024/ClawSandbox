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
	mux.HandleFunc("POST /api/v1/instances/batch-destroy", s.handleBatchDestroyInstances)
	mux.HandleFunc("POST /api/v1/instances/{name}/reset", s.handleResetInstance)
	mux.HandleFunc("GET /api/v1/instances/{name}/logs", s.handleInstanceLogs)
	mux.HandleFunc("POST /api/v1/instances/{name}/configure", s.handleConfigureInstance)
	mux.HandleFunc("GET /api/v1/instances/{name}/configure/status", s.handleConfigureStatus)
	mux.HandleFunc("POST /api/v1/instances/{name}/restart-bot", s.handleRestartBot)
	mux.HandleFunc("GET /api/v1/image/status", s.handleImageStatus)
	mux.HandleFunc("GET /api/v1/image/openclaw-versions", s.handleOpenClawVersions)
	mux.HandleFunc("POST /api/v1/image/build", s.handleImageBuild)
	mux.HandleFunc("POST /api/v1/image/pull", s.handleImagePull)

	// Asset management
	mux.HandleFunc("GET /api/v1/assets/models", s.handleListModelAssets)
	mux.HandleFunc("POST /api/v1/assets/models", s.handleCreateModelAsset)
	mux.HandleFunc("PUT /api/v1/assets/models/{id}", s.handleUpdateModelAsset)
	mux.HandleFunc("DELETE /api/v1/assets/models/{id}", s.handleDeleteModelAsset)
	mux.HandleFunc("POST /api/v1/assets/models/test", s.handleTestModelAsset)
	mux.HandleFunc("GET /api/v1/assets/channels", s.handleListChannelAssets)
	mux.HandleFunc("POST /api/v1/assets/channels", s.handleCreateChannelAsset)
	mux.HandleFunc("PUT /api/v1/assets/channels/{id}", s.handleUpdateChannelAsset)
	mux.HandleFunc("DELETE /api/v1/assets/channels/{id}", s.handleDeleteChannelAsset)
	mux.HandleFunc("POST /api/v1/assets/channels/test", s.handleTestChannelAsset)
	mux.HandleFunc("GET /api/v1/assets/characters", s.handleListCharacterAssets)
	mux.HandleFunc("POST /api/v1/assets/characters", s.handleCreateCharacterAsset)
	mux.HandleFunc("PUT /api/v1/assets/characters/{id}", s.handleUpdateCharacterAsset)
	mux.HandleFunc("DELETE /api/v1/assets/characters/{id}", s.handleDeleteCharacterAsset)

	// Skills (per-instance)
	mux.HandleFunc("GET /api/v1/instances/{name}/skills", s.handleListInstanceSkills)
	mux.HandleFunc("POST /api/v1/instances/{name}/skills/install", s.handleInstallSkill)
	mux.HandleFunc("DELETE /api/v1/instances/{name}/skills/{slug}", s.handleUninstallSkill)
	mux.HandleFunc("GET /api/v1/skills/search", s.handleSearchClawHub)

	// Snapshots
	mux.HandleFunc("GET /api/v1/snapshots", s.handleListSnapshots)
	mux.HandleFunc("POST /api/v1/snapshots", s.handleCreateSnapshot)
	mux.HandleFunc("DELETE /api/v1/snapshots/{id}", s.handleDeleteSnapshot)

	// OAuth
	mux.HandleFunc("POST /api/v1/oauth/codex/start", s.handleCodexOAuthStart)
	mux.HandleFunc("POST /api/v1/oauth/codex/callback", s.handleCodexOAuthCallback)
	mux.HandleFunc("OPTIONS /api/v1/oauth/codex/callback", s.handleCodexOAuthCallback)
	mux.HandleFunc("GET /api/v1/oauth/codex/poll", s.handleCodexOAuthPoll)

	// WebSocket endpoints
	mux.HandleFunc("GET /api/v1/ws/stats", s.handleWSStats)
	mux.HandleFunc("GET /api/v1/ws/logs/{name}", s.handleWSLogs)
	mux.HandleFunc("GET /api/v1/ws/events", s.handleWSEvents)

	// Console reverse proxy (must be before static file handler)
	// Both with and without trailing slash to catch WebSocket connections
	mux.HandleFunc("/console/{name}/", s.handleConsoleProxy)
	mux.HandleFunc("/console/{name}", s.handleConsoleProxy)

	// Static files (frontend)
	staticSub, _ := fs.Sub(StaticFS, "static")
	mux.Handle("/", http.FileServer(http.FS(staticSub)))
}
