package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/weiyong1024/clawsandbox/internal/config"
)

// SnapshotMeta holds metadata for a saved instance snapshot.
type SnapshotMeta struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	SourceInstance string    `json:"source_instance"`
	CreatedAt      time.Time `json:"created_at"`
	SizeBytes      int64     `json:"size_bytes"`
	ModelAssetID   string    `json:"model_asset_id,omitempty"`
}

// SnapshotStore manages snapshot metadata with mutex-protected persistence.
type SnapshotStore struct {
	mu        sync.Mutex
	Snapshots []*SnapshotMeta `json:"snapshots"`
	path      string
}

// LoadSnapshots loads the snapshot store from disk.
func LoadSnapshots() (*SnapshotStore, error) {
	dir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	path := filepath.Join(dir, "snapshots.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SnapshotStore{path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading snapshots: %w", err)
	}
	var s SnapshotStore
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing snapshots: %w", err)
	}
	s.path = path
	return &s, nil
}

// SaveSnapshots persists the snapshot store to disk.
func (s *SnapshotStore) SaveSnapshots() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// List returns a copy of all snapshot metadata.
func (s *SnapshotStore) List() []SnapshotMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SnapshotMeta, len(s.Snapshots))
	for i, snap := range s.Snapshots {
		out[i] = *snap
	}
	return out
}

// Get returns a snapshot by ID, or nil.
func (s *SnapshotStore) Get(id string) *SnapshotMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, snap := range s.Snapshots {
		if snap.ID == id {
			cp := *snap
			return &cp
		}
	}
	return nil
}

// GetByName returns a snapshot by name, or nil.
func (s *SnapshotStore) GetByName(name string) *SnapshotMeta {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, snap := range s.Snapshots {
		if snap.Name == name {
			cp := *snap
			return &cp
		}
	}
	return nil
}

// Add adds a snapshot to the store.
func (s *SnapshotStore) Add(snap *SnapshotMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Snapshots = append(s.Snapshots, snap)
}

// Remove removes a snapshot by ID. Returns false if not found.
func (s *SnapshotStore) Remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, snap := range s.Snapshots {
		if snap.ID == id {
			s.Snapshots = append(s.Snapshots[:i], s.Snapshots[i+1:]...)
			return true
		}
	}
	return false
}
