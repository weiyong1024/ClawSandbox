package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var logsFollow bool

var logsCmd = &cobra.Command{
	Use:     "logs <name>",
	Short:   "View container logs for a claw instance",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox logs claw-1\n  clawsandbox logs claw-1 -f",
	RunE:    runLogs,
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
}

func runLogs(cmd *cobra.Command, args []string) error {
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

	return container.Logs(cli, inst.ContainerID, logsFollow, os.Stdout)
}
