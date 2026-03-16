package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/weiyong1024/clawsandbox/internal/config"
)

// ModelAsset represents a pre-configured LLM provider + API key + model combination.
// Models are shared — multiple instances can use the same model config simultaneously.
type ModelAsset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	Validated bool   `json:"validated"`
}

// ChannelAsset represents a pre-configured messaging channel + token combination.
type ChannelAsset struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Channel   string `json:"channel"`
	Token     string `json:"token"`
	AppID     string `json:"app_id,omitempty"`
	AppSecret string `json:"app_secret,omitempty"`
	Validated bool   `json:"validated"`
	UsedBy    string `json:"used_by"`
}

// CharacterAsset represents a reusable character/persona definition.
// Characters are shared — multiple instances can use the same character simultaneously.
// Fields are rendered into a SOUL.md file and injected into the container.
type CharacterAsset struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Bio        string `json:"bio"`
	Lore       string `json:"lore"`
	Style      string `json:"style"`
	Topics     string `json:"topics"`
	Adjectives string `json:"adjectives"`
}

// AssetStore manages model, channel, and character assets with mutex-protected persistence.
type AssetStore struct {
	mu         sync.Mutex
	Models     []*ModelAsset     `json:"models"`
	Channels   []*ChannelAsset   `json:"channels"`
	Characters []*CharacterAsset `json:"characters"`
	path       string
}

// LoadAssets loads the asset store from disk.
func LoadAssets() (*AssetStore, error) {
	dir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	path := filepath.Join(dir, "assets.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &AssetStore{path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading assets: %w", err)
	}
	var s AssetStore
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing assets: %w", err)
	}
	s.path = path
	return &s, nil
}

// SaveAssets persists the asset store to disk.
func (s *AssetStore) SaveAssets() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// --- Model asset operations ---

// ListModels returns a copy of all model assets.
func (s *AssetStore) ListModels() []ModelAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ModelAsset, len(s.Models))
	for i, m := range s.Models {
		out[i] = *m
	}
	return out
}

// GetModel returns a model asset by ID, or nil.
func (s *AssetStore) GetModel(id string) *ModelAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.Models {
		if m.ID == id {
			cp := *m
			return &cp
		}
	}
	return nil
}

// AddModel adds a model asset.
func (s *AssetStore) AddModel(m *ModelAsset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Models = append(s.Models, m)
}

// UpdateModel replaces a model asset by ID.
func (s *AssetStore) UpdateModel(m *ModelAsset) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Models {
		if existing.ID == m.ID {
			s.Models[i] = m
			return true
		}
	}
	return false
}

// RemoveModel removes a model asset by ID. Returns false if not found.
func (s *AssetStore) RemoveModel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, m := range s.Models {
		if m.ID == id {
			s.Models = append(s.Models[:i], s.Models[i+1:]...)
			return true
		}
	}
	return false
}

// --- Channel asset operations (channels are exclusive — one instance per channel) ---

// ListChannels returns a copy of all channel assets.
func (s *AssetStore) ListChannels() []ChannelAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]ChannelAsset, len(s.Channels))
	for i, c := range s.Channels {
		out[i] = *c
	}
	return out
}

// GetChannel returns a channel asset by ID, or nil.
func (s *AssetStore) GetChannel(id string) *ChannelAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.Channels {
		if c.ID == id {
			cp := *c
			return &cp
		}
	}
	return nil
}

// AddChannel adds a channel asset.
func (s *AssetStore) AddChannel(c *ChannelAsset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Channels = append(s.Channels, c)
}

// UpdateChannel replaces a channel asset by ID.
func (s *AssetStore) UpdateChannel(c *ChannelAsset) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Channels {
		if existing.ID == c.ID {
			s.Channels[i] = c
			return true
		}
	}
	return false
}

// RemoveChannel removes a channel asset by ID. Returns false if not found.
func (s *AssetStore) RemoveChannel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.Channels {
		if c.ID == id {
			s.Channels = append(s.Channels[:i], s.Channels[i+1:]...)
			return true
		}
	}
	return false
}

// AssignChannel marks a channel as used by the given instance.
func (s *AssetStore) AssignChannel(id, instanceName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.Channels {
		if c.ID == id {
			c.UsedBy = instanceName
			return true
		}
	}
	return false
}

// ReleaseChannelByInstance clears UsedBy for all channels assigned to the given instance.
func (s *AssetStore) ReleaseChannelByInstance(instanceName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.Channels {
		if c.UsedBy == instanceName {
			c.UsedBy = ""
		}
	}
}

// --- Character asset operations (characters are shared — multiple instances can use the same character) ---

// ListCharacters returns a copy of all character assets.
func (s *AssetStore) ListCharacters() []CharacterAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CharacterAsset, len(s.Characters))
	for i, c := range s.Characters {
		out[i] = *c
	}
	return out
}

// GetCharacter returns a character asset by ID, or nil.
func (s *AssetStore) GetCharacter(id string) *CharacterAsset {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.Characters {
		if c.ID == id {
			cp := *c
			return &cp
		}
	}
	return nil
}

// AddCharacter adds a character asset.
func (s *AssetStore) AddCharacter(c *CharacterAsset) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Characters = append(s.Characters, c)
}

// UpdateCharacter replaces a character asset by ID.
func (s *AssetStore) UpdateCharacter(c *CharacterAsset) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.Characters {
		if existing.ID == c.ID {
			s.Characters[i] = c
			return true
		}
	}
	return false
}

// RemoveCharacter removes a character asset by ID. Returns false if not found.
func (s *AssetStore) RemoveCharacter(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.Characters {
		if c.ID == id {
			s.Characters = append(s.Characters[:i], s.Characters[i+1:]...)
			return true
		}
	}
	return false
}
