package web

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/clawfleet/clawfleet/internal/state"
)

func TestRuntimeFromImage(t *testing.T) {
	const hermesName = "nousresearch/hermes-agent"

	tests := []struct {
		desc, imageRef, hermesName, want string
	}{
		{"hermes by exact ref", "nousresearch/hermes-agent:latest", hermesName, "hermes"},
		{"hermes case-insensitive", "Nousresearch/Hermes-Agent:1.0", hermesName, "hermes"},
		{"hermes with registry prefix", "docker.io/nousresearch/hermes-agent:dev", hermesName, "hermes"},
		{"openclaw image", "ghcr.io/clawfleet/clawfleet:latest", hermesName, "openclaw"},
		{"unrelated image", "alpine:3.20", hermesName, "openclaw"},
		{"empty hermes name → openclaw", "anything", "", "openclaw"},
		{"empty image → openclaw", "", hermesName, "openclaw"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			if got := runtimeFromImage(tt.imageRef, tt.hermesName); got != tt.want {
				t.Fatalf("runtimeFromImage(%q,%q) = %q, want %q", tt.imageRef, tt.hermesName, got, tt.want)
			}
		})
	}
}

// seedStore writes the given state.json into a fake home dir and returns a
// loaded Store rooted there.
func seedStore(t *testing.T, contents string) *state.Store {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".clawfleet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte(contents), 0o644); err != nil {
		t.Fatalf("write state: %v", err)
	}
	store, err := state.Load()
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	return store
}

func TestBackfillRuntimeTypes_FillsMissing(t *testing.T) {
	store := seedStore(t, `{"instances":[
		{"name":"hermes-1","container_id":"cid-h","status":"running","ports":{"novnc":1,"gateway":2},"created_at":"2026-04-23T21:39:03Z"},
		{"name":"openclaw-1","container_id":"cid-o","status":"running","ports":{"novnc":3,"gateway":4},"created_at":"2026-04-23T21:38:56Z"}
	]}`)

	lookup := func(cid string) (string, error) {
		switch cid {
		case "cid-h":
			return "nousresearch/hermes-agent:latest", nil
		case "cid-o":
			return "ghcr.io/clawfleet/clawfleet:latest", nil
		}
		return "", errors.New("unknown container")
	}

	n, err := backfillRuntimeTypes(store, "nousresearch/hermes-agent", lookup)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n != 2 {
		t.Fatalf("fixed = %d, want 2", n)
	}
	if got := store.Get("hermes-1").RuntimeType; got != "hermes" {
		t.Fatalf("hermes-1 runtime = %q, want hermes", got)
	}
	if got := store.Get("openclaw-1").RuntimeType; got != "openclaw" {
		t.Fatalf("openclaw-1 runtime = %q, want openclaw", got)
	}

	// Reload from disk to verify persistence.
	reloaded, err := state.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got := reloaded.Get("hermes-1").RuntimeType; got != "hermes" {
		t.Fatalf("persisted hermes-1 runtime = %q, want hermes", got)
	}
}

func TestBackfillRuntimeTypes_LeavesPopulatedAlone(t *testing.T) {
	store := seedStore(t, `{"instances":[
		{"name":"already","container_id":"cid","status":"running","ports":{"novnc":1,"gateway":2},"created_at":"2026-04-23T21:39:03Z","runtime_type":"hermes"}
	]}`)

	called := false
	lookup := func(string) (string, error) {
		called = true
		return "", nil
	}

	n, err := backfillRuntimeTypes(store, "nousresearch/hermes-agent", lookup)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n != 0 {
		t.Fatalf("fixed = %d, want 0", n)
	}
	if called {
		t.Fatal("lookup invoked for instance that already had runtime_type")
	}
	if got := store.Get("already").RuntimeType; got != "hermes" {
		t.Fatalf("runtime mutated: %q", got)
	}
}

func TestBackfillRuntimeTypes_SkipsLookupErrors(t *testing.T) {
	store := seedStore(t, `{"instances":[
		{"name":"missing","container_id":"gone","status":"running","ports":{"novnc":1,"gateway":2},"created_at":"2026-04-23T21:39:03Z"}
	]}`)

	lookup := func(string) (string, error) {
		return "", errors.New("no such container")
	}

	n, err := backfillRuntimeTypes(store, "nousresearch/hermes-agent", lookup)
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	if n != 0 {
		t.Fatalf("fixed = %d, want 0", n)
	}
	if got := store.Get("missing").RuntimeType; got != "" {
		t.Fatalf("runtime should remain empty, got %q", got)
	}
}
