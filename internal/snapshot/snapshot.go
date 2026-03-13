package snapshot

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var validName = regexp.MustCompile(`^[\p{L}\p{N}][\p{L}\p{N} _-]{0,63}$`)

// safeDirName converts a display name to a filesystem-safe directory name
// by replacing spaces with hyphens.
func safeDirName(name string) string {
	return strings.ReplaceAll(name, " ", "-")
}

// Save copies an instance's data directory into a named snapshot.
// It skips channels/ and sessions/ directories and strips the "channels"
// key from openclaw.json.
func Save(instanceName, snapshotName string) (*state.SnapshotMeta, error) {
	if !validName.MatchString(snapshotName) {
		return nil, fmt.Errorf("invalid snapshot name %q: use letters, numbers, spaces, hyphens, or underscores (max 64 chars)", snapshotName)
	}

	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}

	dirName := safeDirName(snapshotName)

	srcDir := filepath.Join(dataDir, "data", instanceName, "openclaw")
	if _, err := os.Stat(srcDir); err != nil {
		return nil, fmt.Errorf("instance data not found: %w", err)
	}

	snapshotDir := filepath.Join(dataDir, "snapshots", dirName, "openclaw")
	if _, err := os.Stat(snapshotDir); err == nil {
		return nil, fmt.Errorf("snapshot %q conflicts with an existing snapshot (directory %q already exists)", snapshotName, dirName)
	}

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("creating snapshot dir: %w", err)
	}

	// Copy files, skipping channels/ and sessions/
	var totalSize int64
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(srcDir, path)
		if relPath == "." {
			return nil
		}

		// Skip channels/ and sessions/ directories
		topDir := strings.SplitN(relPath, string(filepath.Separator), 2)[0]
		if topDir == "channels" || topDir == "sessions" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		destPath := filepath.Join(snapshotDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		totalSize += info.Size()
		return copyFile(path, destPath, info.Mode())
	})
	if err != nil {
		// Clean up on failure
		os.RemoveAll(filepath.Join(dataDir, "snapshots", dirName))
		return nil, fmt.Errorf("copying snapshot data: %w", err)
	}

	// Strip "channels" key from openclaw.json in the snapshot
	configPath := filepath.Join(snapshotDir, "openclaw.json")
	if err := stripChannelsFromConfig(configPath); err != nil {
		// Non-fatal: config file might not exist yet
		_ = err
	}

	// Look up model asset ID from state store
	var modelAssetID string
	store, err := state.Load()
	if err == nil {
		if inst := store.Get(instanceName); inst != nil {
			modelAssetID = inst.ModelAssetID
		}
	}

	meta := &state.SnapshotMeta{
		ID:             fmt.Sprintf("snap_%d", time.Now().UnixNano()),
		Name:           snapshotName,
		SourceInstance: instanceName,
		CreatedAt:      time.Now(),
		SizeBytes:      totalSize,
		ModelAssetID:   modelAssetID,
	}

	return meta, nil
}

// Load copies all files from a snapshot directory into an instance data directory.
func Load(snapshotName, instanceDataDir string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	snapshotDir := filepath.Join(dataDir, "snapshots", safeDirName(snapshotName), "openclaw")
	if _, err := os.Stat(snapshotDir); err != nil {
		return fmt.Errorf("snapshot %q not found: %w", snapshotName, err)
	}

	return filepath.Walk(snapshotDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(snapshotDir, path)
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(instanceDataDir, relPath)
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath, info.Mode())
	})
}

// Delete removes a snapshot directory from disk.
func Delete(snapshotName string) error {
	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	snapshotDir := filepath.Join(dataDir, "snapshots", safeDirName(snapshotName))
	if _, err := os.Stat(snapshotDir); err != nil {
		return fmt.Errorf("snapshot %q not found: %w", snapshotName, err)
	}

	return os.RemoveAll(snapshotDir)
}

// SnapshotDir returns the base directory for all snapshots.
func SnapshotDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "snapshots"), nil
}

// stripChannelsFromConfig reads openclaw.json, removes the "channels" key, and writes it back.
func stripChannelsFromConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	delete(cfg, "channels")

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0644)
}

// copyFile copies a single file preserving the given mode.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
