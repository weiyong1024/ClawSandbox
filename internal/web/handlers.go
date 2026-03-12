package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/port"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

// instanceResponse is the JSON representation of a single instance.
type instanceResponse struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	NoVNC          int       `json:"novnc_port"`
	Gateway        int       `json:"gateway_port"`
	CreatedAt      time.Time `json:"created_at"`
	ModelAssetID   string    `json:"model_asset_id,omitempty"`
	ChannelAssetID string    `json:"channel_asset_id,omitempty"`
	ModelName      string    `json:"model_name,omitempty"`
	ChannelName    string    `json:"channel_name,omitempty"`
}

func instanceToResponse(inst state.Instance, assets *state.AssetStore) instanceResponse {
	resp := instanceResponse{
		Name:           inst.Name,
		Status:         inst.Status,
		NoVNC:          inst.Ports.NoVNC,
		Gateway:        inst.Ports.Gateway,
		CreatedAt:      inst.CreatedAt,
		ModelAssetID:   inst.ModelAssetID,
		ChannelAssetID: inst.ChannelAssetID,
	}
	if assets != nil {
		if m := assets.GetModel(inst.ModelAssetID); m != nil {
			resp.ModelName = m.Name
		}
		if c := assets.GetChannel(inst.ChannelAssetID); c != nil {
			resp.ChannelName = c.Name
		}
	}
	return resp
}

// handleListInstances returns all instances with live status from Docker.
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	assets, _ := s.loadAssets()

	instances := store.Snapshot()
	results := make([]instanceResponse, 0, len(instances))
	for _, inst := range instances {
		status, _, _ := container.Status(s.docker, inst.ContainerID)
		store.SetStatus(inst.Name, status)
		inst.Status = status
		results = append(results, instanceToResponse(inst, assets))
	}

	_ = store.Save()
	writeJSON(w, http.StatusOK, map[string]any{"data": results})
}

// createRequest is the JSON body for POST /api/v1/instances.
type createRequest struct {
	Count int `json:"count"`
}

// handleCreateInstances creates N new instances.
func (s *Server) handleCreateInstances(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Count < 1 {
		writeError(w, http.StatusBadRequest, "count must be >= 1")
		return
	}

	cfg := s.config

	exists, err := container.ImageExists(s.docker, cfg.ImageRef())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if !exists {
		writeError(w, http.StatusPreconditionFailed, fmt.Sprintf(
			"Image %s not found. Build the image via System → Image in the Dashboard, or run 'clawsandbox build'.", cfg.ImageRef()))
		return
	}

	if err := container.EnsureNetwork(s.docker); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	memBytes, err := container.ParseMemoryBytes(cfg.Resources.MemoryLimit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	nanoCPUs := int64(cfg.Resources.CPULimit * 1e9)

	dataDir, err := config.DataDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	created := make([]instanceResponse, 0, req.Count)
	for i := 0; i < req.Count; i++ {
		name := store.NextName(cfg.Naming.Prefix)
		usedPorts := store.UsedPorts()

		novncPort, err := port.FindAvailable(cfg.Ports.NoVNCBase, usedPorts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("allocating noVNC port: %v", err))
			return
		}
		usedPorts[novncPort] = true

		gatewayPort, err := port.FindAvailable(cfg.Ports.GatewayBase, usedPorts)
		if err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("allocating gateway port: %v", err))
			return
		}

		instanceDataDir := filepath.Join(dataDir, "data", name, "openclaw")
		if err := os.MkdirAll(instanceDataDir, 0755); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		containerID, err := container.Create(s.docker, container.CreateParams{
			Name:        name,
			ImageRef:    cfg.ImageRef(),
			NoVNCPort:   novncPort,
			GatewayPort: gatewayPort,
			DataDir:     instanceDataDir,
			MemoryBytes: memBytes,
			NanoCPUs:    nanoCPUs,
		})
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if err := container.Start(s.docker, containerID); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("starting %s: %v", name, err))
			return
		}

		inst := &state.Instance{
			Name:        name,
			ContainerID: containerID,
			Status:      "running",
			Ports:       state.Ports{NoVNC: novncPort, Gateway: gatewayPort},
			CreatedAt:   time.Now(),
		}
		store.Add(inst)
		if err := store.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		s.events.Publish(Event{Type: EventCreated, Name: name})
		created = append(created, instanceToResponse(*inst, nil))
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": created})
}

// handleStartInstance starts a stopped instance.
func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := store.Get(name)
	if inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", name))
		return
	}

	if err := container.Start(s.docker, inst.ContainerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	store.SetStatus(name, "running")
	_ = store.Save()

	inst.Status = "running"
	s.events.Publish(Event{Type: EventStarted, Name: name})
	writeJSON(w, http.StatusOK, map[string]any{"data": instanceToResponse(*inst, nil)})
}

// handleStopInstance stops a running instance.
func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := store.Get(name)
	if inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", name))
		return
	}

	if err := container.Stop(s.docker, inst.ContainerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	store.SetStatus(name, "stopped")
	_ = store.Save()

	inst.Status = "stopped"
	s.events.Publish(Event{Type: EventStopped, Name: name})
	writeJSON(w, http.StatusOK, map[string]any{"data": instanceToResponse(*inst, nil)})
}

// handleDestroyInstance removes an instance and optionally purges data.
func (s *Server) handleDestroyInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	// Default: always purge data so recreated instances start fresh.
	// Pass ?purge=false to explicitly preserve data.
	purge := r.URL.Query().Get("purge") != "false"

	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := store.Get(name)
	if inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", name))
		return
	}

	if err := container.Remove(s.docker, inst.ContainerID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	store.Remove(name)
	_ = store.Save()

	// Release any channel assigned to this instance so it becomes available again.
	if assets, err := s.loadAssets(); err == nil {
		assets.ReleaseChannelByInstance(name)
		_ = assets.SaveAssets()
	}

	if purge {
		dataDir, _ := config.DataDir()
		_ = os.RemoveAll(filepath.Join(dataDir, "data", name))
	}

	s.events.Publish(Event{Type: EventDestroyed, Name: name})
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"name": name, "status": "destroyed"}})
}

// handleResetInstance purges the persisted OpenClaw config so the instance
// can be reconfigured from scratch without destroying the container.
func (s *Server) handleResetInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := store.Get(name)
	if inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", name))
		return
	}

	// Stop openclaw gateway if running.
	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status == "running" {
		_ = container.ExecAs(s.docker, inst.ContainerID, "root", []string{
			"supervisorctl", "stop", "openclaw",
		})
	}

	// Remove only OpenClaw config and auth data, preserving Node.js V8 caches
	// and other runtime state to avoid slow JIT recompilation on next configure.
	dataDir, err := config.DataDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	instanceDataDir := filepath.Join(dataDir, "data", name, "openclaw")
	for _, sub := range []string{
		"openclaw.json",
		"agents",
		"sessions",
		"channels",
	} {
		_ = os.RemoveAll(filepath.Join(instanceDataDir, sub))
	}

	// Release any channel assets assigned to this instance.
	assets, err := s.loadAssets()
	if err == nil {
		assets.ReleaseChannelByInstance(name)
		_ = assets.SaveAssets()
	}

	// Restart the container to clear Node.js runtime degradation that causes
	// openclaw CLI commands to hang in long-running containers.
	if status == "running" {
		if err := container.Stop(s.docker, inst.ContainerID); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("restarting %s: %v", name, err))
			return
		}
		if err := container.Start(s.docker, inst.ContainerID); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("restarting %s: %v", name, err))
			return
		}
		store.SetStatus(name, "running")
	}
	_ = store.Save()

	s.events.Publish(Event{Type: EventStopped, Name: name})
	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"name": name, "status": "reset"},
	})
}

// handleInstanceLogs returns the last 100 lines of container logs.
func (s *Server) handleInstanceLogs(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	inst := store.Get(name)
	if inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", name))
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_ = container.Logs(s.docker, inst.ContainerID, false, w)
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    http.StatusText(status),
			"message": msg,
		},
	}); err != nil {
		log.Printf("writeError: %v", err)
	}
}
