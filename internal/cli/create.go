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
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var createCmd = &cobra.Command{
	Use:     "create <N>",
	Short:   "Create N isolated OpenClaw instances",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox create 3\n  clawsandbox create 1",
	RunE:    runCreate,
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
		return fmt.Errorf("image %s not found\nRun 'clawsandbox build' first", cfg.ImageRef())
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

		fmt.Printf("✓  desktop: http://localhost:%d\n", novncPort)
		created++
	}

	fmt.Printf("\n%d claw(s) ready. Run 'clawsandbox desktop %s' to open the desktop.\n",
		created, firstName)
	return nil
}
