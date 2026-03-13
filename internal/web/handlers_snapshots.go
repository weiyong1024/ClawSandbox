package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weiyong1024/clawsandbox/internal/snapshot"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

func (s *Server) loadSnapshots() (*state.SnapshotStore, error) {
	return state.LoadSnapshots()
}

// handleListSnapshots returns all snapshots.
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapStore, err := s.loadSnapshots()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": snapStore.List()})
}

// handleCreateSnapshot saves a new snapshot from an instance.
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		InstanceName string `json:"instance_name"`
		Name         string `json:"name"`
		Description  string `json:"description"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.InstanceName == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "instance_name and name are required")
		return
	}

	// Verify instance exists
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if inst := store.Get(req.InstanceName); inst == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("instance %s not found", req.InstanceName))
		return
	}

	meta, err := snapshot.Save(req.InstanceName, req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("saving snapshot: %v", err))
		return
	}
	meta.Description = req.Description

	snapStore, err := s.loadSnapshots()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	snapStore.Add(meta)
	if err := snapStore.SaveSnapshots(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": meta})
}

// handleDeleteSnapshot removes a snapshot by ID.
func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	snapStore, err := s.loadSnapshots()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	meta := snapStore.Get(id)
	if meta == nil {
		writeError(w, http.StatusNotFound, "snapshot not found")
		return
	}

	if err := snapshot.Delete(meta.Name); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting snapshot: %v", err))
		return
	}

	snapStore.Remove(id)
	if err := snapStore.SaveSnapshots(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"id": id, "status": "deleted"}})
}
