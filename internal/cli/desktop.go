package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var desktopCmd = &cobra.Command{
	Use:     "desktop <name>",
	Short:   "Open a claw's noVNC desktop in the browser",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox desktop claw-1",
	RunE:    runDesktop,
}

func runDesktop(cmd *cobra.Command, args []string) error {
	store, err := state.Load()
	if err != nil {
		return err
	}

	inst := store.Get(args[0])
	if inst == nil {
		return fmt.Errorf("instance not found: %s", args[0])
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}
	status, _, _ := container.Status(cli, inst.ContainerID)
	if status != "running" {
		return fmt.Errorf("%s is not running (status: %s)\nRun 'clawsandbox start %s' first", inst.Name, status, inst.Name)
	}

	url := fmt.Sprintf("http://localhost:%d", inst.Ports.NoVNC)
	fmt.Printf("Opening %s desktop at %s\n", inst.Name, url)
	return openURL(url)
}
