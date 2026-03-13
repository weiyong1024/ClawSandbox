package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "clawsandbox",
	Short: "Deploy and manage a fleet of OpenClaw instances",
	Long: `ClawSandbox lets you spin up multiple isolated OpenClaw instances
on a single machine. Each instance runs in its own Docker container
with a full Linux desktop, accessible via your browser.`,
}

func Execute() {
	rootCmd.AddCommand(
		buildCmd,
		createCmd,
		listCmd,
		startCmd,
		stopCmd,
		restartCmd,
		destroyCmd,
		desktopCmd,
		logsCmd,
		dashboardCmd,
		configCmd,
		configureCmd,
		snapshotCmd,
		versionCmd,
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
