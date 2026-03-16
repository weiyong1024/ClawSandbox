package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weiyong1024/clawsandbox/internal/container"
)

// handleListInstanceSkills returns all skills available in the specified instance.
func (s *Server) handleListInstanceSkills(w http.ResponseWriter, r *http.Request) {
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

	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status != "running" {
		writeError(w, http.StatusPreconditionFailed, "instance must be running to list skills")
		return
	}

	skills, err := container.ListSkills(s.docker, inst.ContainerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": skills})
}

type installSkillRequest struct {
	Slug string `json:"slug"`
}

// handleInstallSkill installs a community skill from ClawHub into an instance.
func (s *Server) handleInstallSkill(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req installSkillRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Slug == "" {
		writeError(w, http.StatusBadRequest, "slug is required")
		return
	}

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

	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status != "running" {
		writeError(w, http.StatusPreconditionFailed, "instance must be running to install skills")
		return
	}

	if err := container.InstallSkill(s.docker, inst.ContainerID, req.Slug); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("install skill: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"status": "installed", "slug": req.Slug},
	})
}

// handleUninstallSkill removes a community skill from an instance.
func (s *Server) handleUninstallSkill(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	slug := r.PathValue("slug")

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

	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status != "running" {
		writeError(w, http.StatusPreconditionFailed, "instance must be running to uninstall skills")
		return
	}

	if err := container.UninstallSkill(s.docker, inst.ContainerID, slug); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("uninstall skill: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{"status": "uninstalled", "slug": slug},
	})
}

// handleSearchClawHub searches the ClawHub marketplace for skills.
// Uses any running instance as the execution context.
func (s *Server) handleSearchClawHub(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	// Find any running instance to execute the search in.
	store, err := s.loadStore()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	instances := store.Snapshot()
	var containerID string
	for _, inst := range instances {
		status, _, _ := container.Status(s.docker, inst.ContainerID)
		if status == "running" {
			containerID = inst.ContainerID
			break
		}
	}
	if containerID == "" {
		writeError(w, http.StatusPreconditionFailed, "no running instance available for search")
		return
	}

	results, err := container.SearchClawHub(s.docker, containerID, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": results})
}
