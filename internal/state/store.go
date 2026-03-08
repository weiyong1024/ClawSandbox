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
	Instances []*Instance `json:"instances"`
	path      string
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
	s.Instances = append(s.Instances, inst)
}

func (s *Store) Remove(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.Instances[:0]
	for _, inst := range s.Instances {
		if inst.Name != name {
			out = append(out, inst)
		}
	}
	s.Instances = out
}

func (s *Store) Get(name string) *Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, inst := range s.Instances {
		if inst.Name == name {
			return inst
		}
	}
	return nil
}

func (s *Store) UsedPorts() map[int]bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	used := make(map[int]bool)
	for _, inst := range s.Instances {
		used[inst.Ports.NoVNC] = true
		used[inst.Ports.Gateway] = true
	}
	return used
}

func (s *Store) NextName(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	used := make(map[string]bool)
	for _, inst := range s.Instances {
		used[inst.Name] = true
	}
	for i := 1; ; i++ {
		name := fmt.Sprintf("%s-%d", prefix, i)
		if !used[name] {
			return name
		}
	}
}
