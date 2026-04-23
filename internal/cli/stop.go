package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/clawfleet/clawfleet/internal/container"
	"github.com/clawfleet/clawfleet/internal/state"
)

var stopCmd = &cobra.Command{
	Use:     "stop <name|all>",
	Short:   "Stop a running instance",
	Args:    cobra.ExactArgs(1),
	Example: "  clawfleet stop hermes-1\n  clawfleet stop all",
	RunE:    runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	store, err := state.Load()
	if err != nil {
		return err
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	targets := resolveTargets(store, args[0])
	if len(targets) == 0 {
		return fmt.Errorf("no instance found: %s", args[0])
	}

	for _, inst := range targets {
		status, _, _ := container.Status(cli, inst.ContainerID)
		if status != "running" {
			fmt.Printf("%s is already stopped, skipping\n", inst.Name)
			store.SetStatus(inst.Name, "stopped")
			continue
		}

		fmt.Printf("Stopping %s ... ", inst.Name)
		if err := container.Stop(cli, inst.ContainerID); err != nil {
			fmt.Println("✗")
			return err
		}
		store.SetStatus(inst.Name, "stopped")
		fmt.Println("✓")
	}

	return store.Save()
}
