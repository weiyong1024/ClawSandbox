package cli

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/clawfleet/clawfleet/internal/container"
	"github.com/clawfleet/clawfleet/internal/state"
)

var shellCmd = &cobra.Command{
	Use:   "shell <name>",
	Short: "Open an interactive shell inside an instance",
	Long: `Opens an interactive terminal session inside a running instance.

For Hermes instances, this launches the Hermes interactive CLI (TUI).
For OpenClaw instances, this opens a bash shell.`,
	Args:    cobra.ExactArgs(1),
	Example: "  clawfleet shell claw-1",
	RunE:    runShell,
}

func runShell(cmd *cobra.Command, args []string) error {
	name := args[0]

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	store, err := state.Load()
	if err != nil {
		return err
	}

	inst := store.Get(name)
	if inst == nil {
		return fmt.Errorf("instance %s not found", name)
	}

	status, _, _ := container.Status(cli, inst.ContainerID)
	if status != "running" {
		return fmt.Errorf("instance %s is not running", name)
	}

	// Build the docker exec command
	var shellArgs []string
	if inst.IsHermes() {
		// Hermes: activate venv and launch interactive TUI
		shellArgs = []string{
			"docker", "exec", "-it", inst.ContainerID,
			"bash", "-c",
			"source /opt/hermes/.venv/bin/activate && exec hermes",
		}
	} else {
		// OpenClaw: open a bash shell as the node user
		shellArgs = []string{
			"docker", "exec", "-it", "-u", "node", inst.ContainerID,
			"bash",
		}
	}

	// Use syscall.Exec to replace the current process so the terminal
	// is fully interactive (stdin/stdout/stderr pass through directly).
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found in PATH: %w", err)
	}

	fmt.Printf("Connecting to %s...\n", name)
	return syscall.Exec(dockerPath, shellArgs, os.Environ())
}
