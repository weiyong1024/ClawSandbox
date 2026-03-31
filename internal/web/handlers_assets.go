package web

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/clawfleet/clawfleet/internal/state"
)

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *Server) loadAssets() (*state.AssetStore, error) {
	return state.LoadAssets()
}

// --- Model Asset Handlers ---

func (s *Server) handleListModelAssets(w http.ResponseWriter, r *http.Request) {
	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	models := store.ListModels()
	// Strip OAuth secrets from the response — the frontend only needs the account ID hint.
	for i := range models {
		models[i].OAuthRefresh = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": models})
}

type createModelRequest struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

func (s *Server) handleCreateModelAsset(w http.ResponseWriter, r *http.Request) {
	var req createModelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Provider == "" || req.APIKey == "" || req.Model == "" {
		writeError(w, http.StatusBadRequest, "provider, api_key, and model are required")
		return
	}

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	name := req.Name
	if name == "" {
		name = fmt.Sprintf("%s %s", providerDisplayName(req.Provider), req.Model)
	}

	asset := &state.ModelAsset{
		ID:        generateID(),
		Name:      name,
		Provider:  req.Provider,
		APIKey:    req.APIKey,
		Model:     req.Model,
		Validated: true, // Only saved after validation passes
	}

	store.AddModel(asset)
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": asset})
}

func (s *Server) handleUpdateModelAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req createModelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	existing := store.GetModel(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "model asset not found")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Provider != "" {
		existing.Provider = req.Provider
	}
	if req.APIKey != "" {
		existing.APIKey = req.APIKey
	}
	if req.Model != "" {
		existing.Model = req.Model
	}
	existing.Validated = true

	if !store.UpdateModel(existing) {
		writeError(w, http.StatusNotFound, "model asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": existing})
}

func (s *Server) handleDeleteModelAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !store.RemoveModel(id) {
		writeError(w, http.StatusNotFound, "model asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

type testModelRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

func (s *Server) handleTestModelAsset(w http.ResponseWriter, r *http.Request) {
	var req testModelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	err := ValidateModelKey(req.Provider, req.APIKey, req.Model)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"valid": false, "error": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"valid": true},
	})
}

// --- Channel Asset Handlers ---

func (s *Server) handleListChannelAssets(w http.ResponseWriter, r *http.Request) {
	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": store.ListChannels()})
}

type createChannelRequest struct {
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	Token     string `json:"token"`
	AppToken  string `json:"app_token"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

func (s *Server) handleCreateChannelAsset(w http.ResponseWriter, r *http.Request) {
	var req createChannelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	asset, err := buildChannelAsset(nil, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	asset.ID = generateID()

	store.AddChannel(asset)
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": asset})
}

func (s *Server) handleUpdateChannelAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req createChannelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	existing := store.GetChannel(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "channel asset not found")
		return
	}

	next, err := buildChannelAsset(existing, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if !store.UpdateChannel(next) {
		writeError(w, http.StatusNotFound, "channel asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": next})
}

func (s *Server) handleDeleteChannelAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !store.RemoveChannel(id) {
		writeError(w, http.StatusNotFound, "channel asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

type testChannelRequest struct {
	Channel   string `json:"channel"`
	Token     string `json:"token"`
	AppToken  string `json:"app_token"`
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`
}

func (s *Server) handleTestChannelAsset(w http.ResponseWriter, r *http.Request) {
	var req testChannelRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	err := ValidateChannelToken(req.Channel, req.Token, req.AppToken, req.AppID, req.AppSecret)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"data": map[string]any{"valid": false, "error": err.Error()},
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"valid": true},
	})
}

// --- Character Asset Handlers ---

func (s *Server) handleListCharacterAssets(w http.ResponseWriter, r *http.Request) {
	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": store.ListCharacters()})
}

type createCharacterRequest struct {
	Name       string `json:"name"`
	Bio        string `json:"bio"`
	Lore       string `json:"lore"`
	Style      string `json:"style"`
	Topics     string `json:"topics"`
	Adjectives string `json:"adjectives"`
}

func (s *Server) handleCreateCharacterAsset(w http.ResponseWriter, r *http.Request) {
	var req createCharacterRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	asset := &state.CharacterAsset{
		ID:         generateID(),
		Name:       req.Name,
		Bio:        req.Bio,
		Lore:       req.Lore,
		Style:      req.Style,
		Topics:     req.Topics,
		Adjectives: req.Adjectives,
	}

	store.AddCharacter(asset)
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": asset})
}

func (s *Server) handleUpdateCharacterAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req createCharacterRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	existing := store.GetCharacter(id)
	if existing == nil {
		writeError(w, http.StatusNotFound, "character asset not found")
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Bio = req.Bio
	existing.Lore = req.Lore
	existing.Style = req.Style
	existing.Topics = req.Topics
	existing.Adjectives = req.Adjectives

	if !store.UpdateCharacter(existing) {
		writeError(w, http.StatusNotFound, "character asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": existing})
}

func (s *Server) handleDeleteCharacterAsset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	store, err := s.loadAssets()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if !store.RemoveCharacter(id) {
		writeError(w, http.StatusNotFound, "character asset not found")
		return
	}
	if err := store.SaveAssets(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "deleted"}})
}

func providerDisplayName(provider string) string {
	switch provider {
	case "anthropic":
		return "Anthropic"
	case "openai":
		return "OpenAI"
	case "openai-codex":
		return "ChatGPT (Codex)"
	case "google":
		return "Google"
	case "deepseek":
		return "DeepSeek"
	default:
		return provider
	}
}

func channelDisplayName(channel string) string {
	switch channel {
	case "telegram":
		return "Telegram"
	case "discord":
		return "Discord"
	case "slack":
		return "Slack"
	case "lark":
		return "Lark"
	default:
		return channel
	}
}

func buildChannelAsset(existing *state.ChannelAsset, req createChannelRequest) (*state.ChannelAsset, error) {
	asset := &state.ChannelAsset{}
	if existing != nil {
		*asset = *existing
	}

	channelChanged := existing != nil && req.Channel != "" && req.Channel != existing.Channel
	if existing == nil || channelChanged {
		asset.Token = ""
		asset.AppToken = ""
		asset.AppID = ""
		asset.AppSecret = ""
	}

	if req.Name != "" {
		asset.Name = req.Name
	}
	if req.Channel != "" {
		asset.Channel = req.Channel
	}
	if asset.Channel == "" {
		return nil, fmt.Errorf("channel is required")
	}

	if req.Token != "" || existing == nil || channelChanged {
		asset.Token = req.Token
	}
	if req.AppToken != "" || existing == nil || channelChanged {
		asset.AppToken = req.AppToken
	}
	if req.AppID != "" || existing == nil || channelChanged {
		asset.AppID = req.AppID
	}
	if req.AppSecret != "" || existing == nil || channelChanged {
		asset.AppSecret = req.AppSecret
	}

	switch asset.Channel {
	case "telegram", "discord":
		asset.AppToken = ""
		asset.AppID = ""
		asset.AppSecret = ""
	case "slack":
		asset.AppID = ""
		asset.AppSecret = ""
	case "lark":
		asset.Token = ""
		asset.AppToken = ""
	default:
		return nil, fmt.Errorf("unsupported channel: %s", asset.Channel)
	}

	if err := ValidateChannelCredentials(
		asset.Channel,
		asset.Token,
		asset.AppToken,
		asset.AppID,
		asset.AppSecret,
	); err != nil {
		return nil, err
	}

	if asset.Name == "" {
		asset.Name = fmt.Sprintf("%s Bot", channelDisplayName(asset.Channel))
	}
	asset.Validated = true
	return asset, nil
}
