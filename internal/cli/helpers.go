package cli

import (
	"github.com/weiyong1024/clawsandbox/internal/state"
)

// resolveTargets returns copies of the instances matching name, or all instances if name == "all".
func resolveTargets(store *state.Store, name string) []state.Instance {
	if name == "all" {
		return store.Snapshot()
	}
	inst := store.Get(name)
	if inst == nil {
		return nil
	}
	return []state.Instance{*inst}
}
