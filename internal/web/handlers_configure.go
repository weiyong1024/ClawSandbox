package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/weiyong1024/clawsandbox/internal/container"
)

// configureRequest is the JSON body for POST /api/v1/instances/{name}/configure.
// Supports both asset-based (model_asset_id/channel_asset_id) and direct field configuration.
type configureRequest struct {
	// Asset-based configuration
	ModelAssetID   string `json:"model_asset_id"`
	ChannelAssetID string `json:"channel_asset_id"`

	// Direct configuration (legacy, still supported)
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	Channel      string `json:"channel"`
	ChannelToken string `json:"channel_token"`
	AppID        string `json:"app_id"`
	AppSecret    string `json:"app_secret"`
}

// handleConfigureInstance configures an OpenClaw instance via docker exec.
func (s *Server) handleConfigureInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req configureRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// If asset IDs are provided, resolve them to actual config values
	if req.ModelAssetID != "" {
		assets, err := s.loadAssets()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		model := assets.GetModel(req.ModelAssetID)
		if model == nil {
			writeError(w, http.StatusBadRequest, "model asset not found")
			return
		}
		if !model.Validated {
			writeError(w, http.StatusBadRequest, "model asset has not been validated")
			return
		}

		req.Provider = model.Provider
		req.APIKey = model.APIKey
		req.Model = model.Model

		// Handle channel asset
		if req.ChannelAssetID != "" {
			channel := assets.GetChannel(req.ChannelAssetID)
			if channel == nil {
				writeError(w, http.StatusBadRequest, "channel asset not found")
				return
			}
			if !channel.Validated {
				writeError(w, http.StatusBadRequest, "channel asset has not been validated")
				return
			}
			req.Channel = channel.Channel
			req.ChannelToken = channel.Token
			req.AppID = channel.AppID
			req.AppSecret = channel.AppSecret
		}

		// Channel is exclusive — release previous and assign new
		assets.ReleaseChannelByInstance(name)
		if req.ChannelAssetID != "" {
			assets.AssignChannel(req.ChannelAssetID, name)
		}

		if err := assets.SaveAssets(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if req.Provider == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "provider and api_key are required")
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

	// Ensure instance is running
	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status != "running" {
		if err := container.Start(s.docker, inst.ContainerID); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("starting instance: %v", err))
			return
		}
		store.SetStatus(name, "running")
		_ = store.Save()
	}

	// Resolve bot display name from the channel platform for text @mention detection.
	// Lark/Feishu doesn't support programmatic bot name resolution via API,
	// so we skip it — text @mention detection is not needed for Feishu
	// (it uses native platform mentions).
	var botName string
	if req.Channel != "" && req.Channel != "lark" && req.ChannelToken != "" {
		botName = resolveBotName(req.Channel, req.ChannelToken)
	}

	if err := container.Configure(s.docker, container.ConfigureParams{
		ContainerID:  inst.ContainerID,
		Provider:     req.Provider,
		APIKey:       req.APIKey,
		Model:        req.Model,
		Channel:      req.Channel,
		ChannelToken: req.ChannelToken,
		AppID:        req.AppID,
		AppSecret:    req.AppSecret,
		BotName:      botName,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("configure failed: %v", err))
		return
	}

	// Persist which asset IDs were used so the card and dialog can show them.
	store.SetConfig(name, req.ModelAssetID, req.ChannelAssetID)
	_ = store.Save()

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"status":  "configured",
			"message": fmt.Sprintf("Instance %s configured successfully", name),
		},
	})
}

// handleConfigureStatus returns the configuration status of an instance.
func (s *Server) handleConfigureStatus(w http.ResponseWriter, r *http.Request) {
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
		writeJSON(w, http.StatusOK, map[string]any{
			"data": &container.ConfigInfo{Configured: false},
		})
		return
	}

	info, err := container.ConfigStatus(s.docker, inst.ContainerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": info})
}
