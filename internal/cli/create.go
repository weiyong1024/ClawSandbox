package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/port"
	"github.com/weiyong1024/clawsandbox/internal/snapshot"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var pullFlag bool
var fromSnapshotFlag string

var createCmd = &cobra.Command{
	Use:     "create <N>",
	Short:   "Create N isolated OpenClaw instances",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox create 3\n  clawsandbox create 1\n  clawsandbox create 3 --pull",
	RunE:    runCreate,
}

func init() {
	createCmd.Flags().BoolVar(&pullFlag, "pull", false, "Force re-pull image even if found locally")
	createCmd.Flags().StringVar(&fromSnapshotFlag, "from-snapshot", "", "Create instance from a saved snapshot")
}

func runCreate(cmd *cobra.Command, args []string) error {
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 {
		return fmt.Errorf("N must be a positive integer")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	// Check image exists
	exists, err := container.ImageExists(cli, cfg.ImageRef())
	if err != nil {
		return err
	}
	if !exists {
		if pullFlag {
			fmt.Printf("Image %s not found locally, pulling from registry...\n", cfg.ImageRef())
			if pullErr := container.PullImage(cli, cfg.Image.Name, cfg.Image.Tag, os.Stdout); pullErr != nil {
				return fmt.Errorf("pull failed: %v\nRun 'clawsandbox build' to build it manually", pullErr)
			}
			fmt.Println("Image pulled successfully.")
		} else {
			return fmt.Errorf("Image %s not found. Run 'clawsandbox build' or build via Dashboard.\nUse --pull to pull from the registry instead.", cfg.ImageRef())
		}
	}

	// Ensure network
	if err := container.EnsureNetwork(cli); err != nil {
		return err
	}

	// Load state
	store, err := state.Load()
	if err != nil {
		return err
	}

	// Parse resource limits
	memBytes, err := container.ParseMemoryBytes(cfg.Resources.MemoryLimit)
	if err != nil {
		return err
	}
	nanoCPUs := int64(cfg.Resources.CPULimit * 1e9)

	dataDir, err := config.DataDir()
	if err != nil {
		return err
	}

	created := 0
	firstName := ""
	for i := 0; i < n; i++ {
		name := store.NextName(cfg.Naming.Prefix)
		if firstName == "" {
			firstName = name
		}
		usedPorts := store.UsedPorts()

		novncPort, err := port.FindAvailable(cfg.Ports.NoVNCBase, usedPorts)
		if err != nil {
			return fmt.Errorf("allocating noVNC port: %w", err)
		}
		usedPorts[novncPort] = true

		gatewayPort, err := port.FindAvailable(cfg.Ports.GatewayBase, usedPorts)
		if err != nil {
			return fmt.Errorf("allocating gateway port: %w", err)
		}

		instanceDataDir := filepath.Join(dataDir, "data", name, "openclaw")
		if err := os.MkdirAll(instanceDataDir, 0755); err != nil {
			return fmt.Errorf("creating data dir for %s: %w", name, err)
		}

		// Load snapshot data if specified
		if fromSnapshotFlag != "" {
			if err := snapshot.Load(fromSnapshotFlag, instanceDataDir); err != nil {
				return fmt.Errorf("loading snapshot: %w", err)
			}
		}

		fmt.Printf("Creating %s ... ", name)

		containerID, err := container.Create(cli, container.CreateParams{
			Name:        name,
			ImageRef:    cfg.ImageRef(),
			NoVNCPort:   novncPort,
			GatewayPort: gatewayPort,
			DataDir:     instanceDataDir,
			MemoryBytes: memBytes,
			NanoCPUs:    nanoCPUs,
		})
		if err != nil {
			fmt.Println("✗")
			return err
		}

		if err := container.Start(cli, containerID); err != nil {
			fmt.Println("✗")
			return fmt.Errorf("starting %s: %w", name, err)
		}

		inst := &state.Instance{
			Name:        name,
			ContainerID: containerID,
			Status:      "running",
			Ports:       state.Ports{NoVNC: novncPort, Gateway: gatewayPort},
			CreatedAt:   time.Now(),
		}
		store.Add(inst)
		if err := store.Save(); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}

		// Associate model asset from snapshot if available
		if fromSnapshotFlag != "" {
			if snapStore, err := state.LoadSnapshots(); err == nil {
				if snapMeta := snapStore.GetByName(fromSnapshotFlag); snapMeta != nil && snapMeta.ModelAssetID != "" {
					store.SetConfig(name, snapMeta.ModelAssetID, "", "")
					_ = store.Save()
				}
			}
		}

		fmt.Printf("✓  desktop: http://localhost:%d\n", novncPort)
		created++
	}

	fmt.Printf("\n%d claw(s) ready. Run 'clawsandbox desktop %s' to open the desktop.\n",
		created, firstName)
	return nil
}
