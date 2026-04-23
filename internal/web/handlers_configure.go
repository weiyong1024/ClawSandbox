package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/clawfleet/clawfleet/internal/container"
	"github.com/clawfleet/clawfleet/internal/state"
)

// configureRequest is the JSON body for POST /api/v1/instances/{name}/configure.
// Supports both asset-based (model_asset_id/channel_asset_id) and direct field configuration.
type configureRequest struct {
	// Asset-based configuration
	ModelAssetID     string `json:"model_asset_id"`
	ChannelAssetID   string `json:"channel_asset_id"`
	CharacterAssetID string `json:"character_asset_id"`

	// Direct configuration (legacy, still supported)
	Provider        string `json:"provider"`
	APIKey          string `json:"api_key"`
	Model           string `json:"model"`
	Channel         string `json:"channel"`
	ChannelToken    string `json:"channel_token"`
	ChannelAppToken string `json:"channel_app_token"`
	AppID           string `json:"app_id"`
	AppSecret       string `json:"app_secret"`

	// OAuth fields (populated from model asset for openai-codex)
	OAuthRefresh   string `json:"-"`
	OAuthExpires   int64  `json:"-"`
	OAuthAccountID string `json:"-"`
}

// handleConfigureInstance configures an OpenClaw instance via docker exec.
func (s *Server) handleConfigureInstance(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var req configureRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var assetsStore *state.AssetStore
	if req.ModelAssetID != "" || req.ChannelAssetID != "" {
		// Resolve asset IDs to actual config values.
		assets, err := s.loadAssets()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if req.ModelAssetID != "" {
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
			req.OAuthRefresh = model.OAuthRefresh
			req.OAuthExpires = model.OAuthExpires
			req.OAuthAccountID = model.OAuthAccountID
		}

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
			req.ChannelAppToken = channel.AppToken
			req.AppID = channel.AppID
			req.AppSecret = channel.AppSecret
		}
		assetsStore = assets
	}

	// For OpenClaw, model (provider + api_key) is required.
	// For Hermes, model is optional — user may configure only a channel
	// (model may already be set via the Hermes native Dashboard).
	if req.ModelAssetID == "" && req.Provider == "" {
		// No model asset and no direct provider — check if this is a Hermes channel-only config
		store0, err := s.loadStore()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		inst0 := store0.Get(name)
		if inst0 == nil || !inst0.IsHermes() {
			writeError(w, http.StatusBadRequest, "provider and api_key are required")
			return
		}
		// Hermes channel-only: must have a channel
		if req.ChannelAssetID == "" && req.Channel == "" {
			writeError(w, http.StatusBadRequest, "either model or channel is required")
			return
		}
	} else if req.Provider == "" || req.APIKey == "" {
		writeError(w, http.StatusBadRequest, "provider and api_key are required")
		return
	}
	if req.Channel != "" {
		if err := ValidateChannelCredentials(
			req.Channel,
			req.ChannelToken,
			req.ChannelAppToken,
			req.AppID,
			req.AppSecret,
		); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
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
	// Ensure instance is running before configuration.
	status, _, _ := container.Status(s.docker, inst.ContainerID)
	if status != "running" {
		if err := container.Start(s.docker, inst.ContainerID); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("starting instance: %v", err))
			return
		}
		store.SetStatus(name, "running")
		_ = store.Save()
	}

	if inst.IsHermes() {
		// Hermes supports discord, telegram, and slack channels.
		if req.Channel != "" && req.Channel != "discord" && req.Channel != "telegram" && req.Channel != "slack" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Hermes does not support %s channel", req.Channel))
			return
		}

		if err := container.ConfigureHermes(s.docker, container.HermesConfigureParams{
			ContainerID:    inst.ContainerID,
			Provider:       req.Provider,
			APIKey:         req.APIKey,
			Model:          req.Model,
			Channel:        req.Channel,
			ChannelToken:   req.ChannelToken,
			OAuthRefresh:   req.OAuthRefresh,
			OAuthExpires:   req.OAuthExpires,
			OAuthAccountID: req.OAuthAccountID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("configure failed: %v", err))
			return
		}
	} else {
		// OpenClaw configuration path.

		// Resolve bot display name from the channel platform for text @mention detection.
		// Lark/Feishu doesn't support programmatic bot name resolution via API,
		// so we skip it — text @mention detection is not needed for Feishu
		// (it uses native platform mentions).
		var botName string
		if req.Channel != "" && req.Channel != "lark" && req.ChannelToken != "" {
			botName = resolveBotName(req.Channel, req.ChannelToken)
		}

		// Resolve character asset into SoulParams so it's injected before gateway starts.
		// Include roster (teammates) so the bot knows about its fleet.
		var soul *container.SoulParams
		if req.CharacterAssetID != "" {
			assets, loadErr := s.loadAssets()
			if loadErr == nil {
				if ch := assets.GetCharacter(req.CharacterAssetID); ch != nil {
					soul = &container.SoulParams{
						Name:       ch.Name,
						Bio:        ch.Bio,
						Lore:       ch.Lore,
						Style:      ch.Style,
						Topics:     ch.Topics,
						Adjectives: ch.Adjectives,
						Teammates:  buildRoster(name, store, assets),
					}
				}
			}
		}

		if err := container.Configure(s.docker, container.ConfigureParams{
			ContainerID:     inst.ContainerID,
			Provider:        req.Provider,
			APIKey:          req.APIKey,
			Model:           req.Model,
			Channel:         req.Channel,
			ChannelToken:    req.ChannelToken,
			ChannelAppToken: req.ChannelAppToken,
			AppID:           req.AppID,
			AppSecret:       req.AppSecret,
			BotName:         botName,
			Soul:            soul,
			OAuthRefresh:    req.OAuthRefresh,
			OAuthExpires:    req.OAuthExpires,
			OAuthAccountID:  req.OAuthAccountID,
		}); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("configure failed: %v", err))
			return
		}

		// Refresh teammates' SOUL.md so their roster includes this instance.
		if req.CharacterAssetID != "" {
			if refreshAssets, err := s.loadAssets(); err == nil {
				if refreshStore, err := s.loadStore(); err == nil {
					s.refreshTeammateRosters(name, refreshStore, refreshAssets)
				}
			}
		}
	}

	// Persist which asset IDs were used so the card and dialog can show them.
	store.SetConfig(name, req.ModelAssetID, req.ChannelAssetID, req.CharacterAssetID)
	_ = store.Save()

	if assetsStore != nil {
		assetsStore.ReleaseChannelByInstance(name)
		if req.ChannelAssetID != "" {
			assetsStore.AssignChannel(req.ChannelAssetID, name)
		}
		if err := assetsStore.SaveAssets(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]string{
			"status":  "configured",
			"message": fmt.Sprintf("Instance %s configured successfully", name),
		},
	})
}

// buildRoster returns the list of teammates for the given instance, excluding
// itself. Only instances with a character asset are included in the roster.
func buildRoster(excludeName string, store *state.Store, assets *state.AssetStore) []container.Teammate {
	instances := store.Snapshot()
	var teammates []container.Teammate
	for _, inst := range instances {
		if inst.Name == excludeName {
			continue
		}
		if inst.CharacterAssetID == "" {
			continue
		}
		ch := assets.GetCharacter(inst.CharacterAssetID)
		if ch == nil {
			continue
		}
		channelName := ""
		if inst.ChannelAssetID != "" {
			if ca := assets.GetChannel(inst.ChannelAssetID); ca != nil {
				channelName = ca.Channel
			}
		}
		teammates = append(teammates, container.Teammate{
			Name:    ch.Name,
			Bio:     ch.Bio,
			Channel: channelName,
		})
	}
	return teammates
}

// refreshTeammateRosters rewrites SOUL.md for all other running instances that
// have a character, so their roster reflects the latest fleet state.
// Errors are logged but do not fail the calling operation.
func (s *Server) refreshTeammateRosters(excludeName string, store *state.Store, assets *state.AssetStore) {
	instances := store.Snapshot()
	for _, inst := range instances {
		if inst.Name == excludeName {
			continue
		}
		if inst.CharacterAssetID == "" {
			continue
		}
		if inst.Status != "running" {
			continue
		}
		ch := assets.GetCharacter(inst.CharacterAssetID)
		if ch == nil {
			continue
		}
		teammates := buildRoster(inst.Name, store, assets)
		soul := container.SoulParams{
			Name:       ch.Name,
			Bio:        ch.Bio,
			Lore:       ch.Lore,
			Style:      ch.Style,
			Topics:     ch.Topics,
			Adjectives: ch.Adjectives,
			Teammates:  teammates,
		}
		if err := container.InjectSoul(s.docker, inst.ContainerID, soul); err != nil {
			log.Printf("roster refresh %s: %v", inst.Name, err)
		}
	}
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
