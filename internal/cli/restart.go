package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/clawfleet/clawfleet/internal/container"
	"github.com/clawfleet/clawfleet/internal/state"
)

var restartCmd = &cobra.Command{
	Use:     "restart <name|all>",
	Short:   "Restart an instance (stop then start)",
	Args:    cobra.ExactArgs(1),
	Example: "  clawfleet restart openclaw-1\n  clawfleet restart all",
	RunE:    runRestart,
}

func runRestart(cmd *cobra.Command, args []string) error {
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
		fmt.Printf("Restarting %s ... ", inst.Name)

		status, _, _ := container.Status(cli, inst.ContainerID)
		if status == "running" {
			if err := container.Stop(cli, inst.ContainerID); err != nil {
				fmt.Println("✗")
				return fmt.Errorf("stopping %s: %w", inst.Name, err)
			}
		}

		if err := container.Start(cli, inst.ContainerID); err != nil {
			fmt.Println("✗")
			return fmt.Errorf("starting %s: %w", inst.Name, err)
		}

		store.SetStatus(inst.Name, "running")
		fmt.Println("✓")
	}

	return store.Save()
}
