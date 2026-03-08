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

type Ports struct {
	NoVNC   int `json:"novnc"`
	Gateway int `json:"gateway"`
}

type Instance struct {
	Name        string    `json:"name"`
	ContainerID string    `json:"container_id"`
	Status      string    `json:"status"`
	Ports       Ports     `json:"ports"`
	CreatedAt   time.Time `json:"created_at"`
}

type Store struct {
	mu        sync.Mutex
	instances []*Instance
	path      string
}

// MarshalJSON implements json.Marshaler to serialize the unexported instances field.
func (s *Store) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Instances []*Instance `json:"instances"`
	}{Instances: s.instances})
}

// UnmarshalJSON implements json.Unmarshaler to deserialize into the unexported instances field.
func (s *Store) UnmarshalJSON(data []byte) error {
	var raw struct {
		Instances []*Instance `json:"instances"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.instances = raw.Instances
	return nil
}

func Load() (*Store, error) {
	dir, err := config.DataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating data dir: %w", err)
	}
	path := filepath.Join(dir, "state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{path: path}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading state: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing state: %w", err)
	}
	s.path = path
	return &s, nil
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) Add(inst *Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances = append(s.instances, inst)
}

func (s *Store) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Instance, 0, len(s.instances))
	for _, inst := range s.instances {
		if inst.Name != name {
			out = append(out, inst)
		}
	}
	s.instances = out
}

// Get returns a copy of the instance with the given name, or nil if not found.
func (s *Store) Get(name string) *Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, inst := range s.instances {
		if inst.Name == name {
			cp := *inst
			return &cp
		}
	}
	return nil
}

// SetStatus updates the status of the named instance.
func (s *Store) SetStatus(name, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, inst := range s.instances {
		if inst.Name == name {
			inst.Status = status
			return
		}
	}
}

// Snapshot returns a copy of all instances.
func (s *Store) Snapshot() []Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Instance, len(s.instances))
	for i, inst := range s.instances {
		out[i] = *inst
	}
	return out
}

func (s *Store) UsedPorts() map[int]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	used := make(map[int]bool)
	for _, inst := range s.instances {
		used[inst.Ports.NoVNC] = true
		used[inst.Ports.Gateway] = true
	}
	return used
}

func (s *Store) NextName(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	used := make(map[string]bool)
	for _, inst := range s.instances {
		used[inst.Name] = true
	}
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%d", prefix, i)
		if !used[name] {
			return name
		}
	}
}
