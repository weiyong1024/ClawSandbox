package web

import (
	"testing"

	"github.com/clawfleet/clawfleet/internal/state"
)

// mockStoreForRoster creates a Store with the given instances for roster testing.
// It uses a temp file so Save() doesn't fail.
func mockStoreForRoster(t *testing.T, instances []*state.Instance) *state.Store {
	t.Helper()
	store, err := state.Load()
	if err != nil {
		t.Fatalf("loading store: %v", err)
	}
	for _, inst := range instances {
		store.Add(inst)
	}
	return store
}

func mockAssetsForRoster(t *testing.T) *state.AssetStore {
	t.Helper()
	assets, err := state.LoadAssets()
	if err != nil {
		t.Fatalf("loading assets: %v", err)
	}
	assets.AddCharacter(&state.CharacterAsset{
		ID:   "char-zeus",
		Name: "Zeus",
		Bio:  "King of the Olympian gods.",
	})
	assets.AddCharacter(&state.CharacterAsset{
		ID:   "char-wukong",
		Name: "SunWukong",
		Bio:  "The Monkey King who defied heaven.",
	})
	assets.AddCharacter(&state.CharacterAsset{
		ID:   "char-odin",
		Name: "Odin",
		Bio:  "The All-Father of Norse mythology.",
	})
	assets.AddChannel(&state.ChannelAsset{
		ID:      "ch-discord",
		Name:    "team-discord",
		Channel: "discord",
	})
	return assets
}

func TestBuildRosterExcludesSelf(t *testing.T) {
	assets := mockAssetsForRoster(t)
	store := mockStoreForRoster(t, []*state.Instance{
		{Name: "zeus-1", ContainerID: "c1", Status: "running", CharacterAssetID: "char-zeus", ChannelAssetID: "ch-discord"},
		{Name: "wukong-1", ContainerID: "c2", Status: "running", CharacterAssetID: "char-wukong", ChannelAssetID: "ch-discord"},
		{Name: "odin-1", ContainerID: "c3", Status: "running", CharacterAssetID: "char-odin", ChannelAssetID: "ch-discord"},
	})

	roster := buildRoster("zeus-1", store, assets)

	if len(roster) != 2 {
		t.Fatalf("expected 2 teammates, got %d", len(roster))
	}
	for _, tm := range roster {
		if tm.Name == "Zeus" {
			t.Fatal("roster must not include the excluded instance's character")
		}
	}
}

func TestBuildRosterExcludesNoCharacter(t *testing.T) {
	assets := mockAssetsForRoster(t)
	store := mockStoreForRoster(t, []*state.Instance{
		{Name: "zeus-1", ContainerID: "c1", Status: "running", CharacterAssetID: "char-zeus"},
		{Name: "bare-1", ContainerID: "c2", Status: "running", CharacterAssetID: ""},
	})

	roster := buildRoster("zeus-1", store, assets)

	if len(roster) != 0 {
		t.Fatalf("expected 0 teammates (bare instance excluded), got %d", len(roster))
	}
}

func TestBuildRosterIncludesStoppedInstances(t *testing.T) {
	assets := mockAssetsForRoster(t)
	store := mockStoreForRoster(t, []*state.Instance{
		{Name: "zeus-1", ContainerID: "c1", Status: "running", CharacterAssetID: "char-zeus"},
		{Name: "wukong-1", ContainerID: "c2", Status: "stopped", CharacterAssetID: "char-wukong", ChannelAssetID: "ch-discord"},
	})

	roster := buildRoster("zeus-1", store, assets)

	if len(roster) != 1 {
		t.Fatalf("expected 1 teammate (stopped instance included), got %d", len(roster))
	}
	if roster[0].Name != "SunWukong" {
		t.Fatalf("expected SunWukong, got %s", roster[0].Name)
	}
	if roster[0].Channel != "discord" {
		t.Fatalf("expected discord channel, got %s", roster[0].Channel)
	}
}
