package cli

import "github.com/spf13/cobra"

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage instance snapshots",
	Long:  "Save, list, and delete snapshots of configured instances.",
}

func init() {
	snapshotCmd.AddCommand(snapshotSaveCmd, snapshotListCmd, snapshotDeleteCmd)
}
