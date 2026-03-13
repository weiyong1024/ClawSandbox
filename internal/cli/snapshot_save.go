package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/snapshot"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var (
	snapshotSaveName string
	snapshotSaveDesc string
)

var snapshotSaveCmd = &cobra.Command{
	Use:     "save <instance-name>",
	Short:   "Save a snapshot of an instance",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox snapshot save claw-1\n  clawsandbox snapshot save claw-1 --name my-snapshot --description \"Production config\"",
	RunE:    runSnapshotSave,
}

func init() {
	snapshotSaveCmd.Flags().StringVar(&snapshotSaveName, "name", "", "Snapshot name (default: <instance>-snap-<timestamp>)")
	snapshotSaveCmd.Flags().StringVar(&snapshotSaveDesc, "description", "", "Optional description")
}

func runSnapshotSave(cmd *cobra.Command, args []string) error {
	instanceName := args[0]

	// Verify instance exists
	store, err := state.Load()
	if err != nil {
		return err
	}
	if inst := store.Get(instanceName); inst == nil {
		return fmt.Errorf("instance %s not found", instanceName)
	}

	// Default snapshot name
	name := snapshotSaveName
	if name == "" {
		name = fmt.Sprintf("%s-snap-%s", instanceName, time.Now().Format("20060102-150405"))
	}

	fmt.Printf("Saving snapshot %q from %s ... ", name, instanceName)

	meta, err := snapshot.Save(instanceName, name)
	if err != nil {
		fmt.Println("✗")
		return err
	}

	meta.Description = snapshotSaveDesc

	// Persist metadata
	snapStore, err := state.LoadSnapshots()
	if err != nil {
		fmt.Println("✗")
		return err
	}
	snapStore.Add(meta)
	if err := snapStore.SaveSnapshots(); err != nil {
		fmt.Println("✗")
		return err
	}

	fmt.Printf("✓  (%d bytes)\n", meta.SizeBytes)
	return nil
}
