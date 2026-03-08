package web

import (
	"encoding/json"
	"fmt"
	"io"
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
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	NoVNC     int       `json:"novnc_port"`
	Gateway   int       `json:"gateway_port"`
	CreatedAt time.Time `json:"created_at"`
}

func instanceToResponse(inst state.Instance) instanceResponse {
	return instanceResponse{
		Name:      inst.Name,
		Status:    inst.Status,
		NoVNC:     inst.Ports.NoVNC,
		Gateway:   inst.Ports.Gateway,
		CreatedAt: inst.CreatedAt,
	}
}

// handleListInstances returns all instances with live status from Docker.
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	instances := store.Snapshot()
	results := make([]instanceResponse, 0, len(instances))
	for _, inst := range instances {
		status, _, _ := container.Status(s.docker, inst.ContainerID)
		store.SetStatus(inst.Name, status)
		inst.Status = status
		results = append(results, instanceToResponse(inst))
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
		log.Printf("Image %s not found locally, attempting pull from registry...", cfg.ImageRef())
		if pullErr := container.PullImage(s.docker, cfg.Image.Name, cfg.Image.Tag, io.Discard); pullErr != nil {
			writeError(w, http.StatusPreconditionFailed, fmt.Sprintf(
				"image %s not found locally and pull failed: %v — run 'clawsandbox build'", cfg.ImageRef(), pullErr))
			return
		}
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
		created = append(created, instanceToResponse(*inst))
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
	writeJSON(w, http.StatusOK, map[string]any{"data": instanceToResponse(*inst)})
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
	writeJSON(w, http.StatusOK, map[string]any{"data": instanceToResponse(*inst)})
}

// handleDestroyInstance removes an instance and optionally purges data.
func (s *Server) handleDestroyInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	purge := r.URL.Query().Get("purge") == "true"

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

	if purge {
		dataDir, _ := config.DataDir()
		_ = os.RemoveAll(filepath.Join(dataDir, "data", name))
	}

	s.events.Publish(Event{Type: EventDestroyed, Name: name})
	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"name": name, "status": "destroyed"}})
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
