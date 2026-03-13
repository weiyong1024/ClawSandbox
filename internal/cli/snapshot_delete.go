package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/snapshot"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var snapshotDeleteCmd = &cobra.Command{
	Use:     "delete <snapshot-name>",
	Short:   "Delete a snapshot",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox snapshot delete my-snapshot",
	RunE:    runSnapshotDelete,
}

func runSnapshotDelete(cmd *cobra.Command, args []string) error {
	name := args[0]

	snapStore, err := state.LoadSnapshots()
	if err != nil {
		return err
	}

	meta := snapStore.GetByName(name)
	if meta == nil {
		return fmt.Errorf("snapshot %q not found", name)
	}

	if err := snapshot.Delete(name); err != nil {
		return err
	}

	snapStore.Remove(meta.ID)
	if err := snapStore.SaveSnapshots(); err != nil {
		return err
	}

	fmt.Printf("Snapshot %q deleted.\n", name)
	return nil
}
