package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dashboardRestartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "Restart the Dashboard server (stop then serve)",
	Example: "  clawfleet dashboard restart\n  clawfleet dashboard restart --port 9090",
	RunE:    runDashboardRestart,
}

func init() {
	dashboardRestartCmd.Flags().IntVar(&dashboardServePort, "port", 8080, "HTTP listen port")
	dashboardRestartCmd.Flags().StringVar(&dashboardServeHost, "host", "127.0.0.1", "HTTP listen host")
}

func runDashboardRestart(cmd *cobra.Command, args []string) error {
	// Try service manager first
	mgr := NewServiceManager()
	if installed, _ := mgr.IsInstalled(); installed {
		fmt.Printf("Restarting Dashboard daemon ... ")
		if err := mgr.Restart(); err != nil {
			return fmt.Errorf("failed to restart: %w", err)
		}
		fmt.Println("done")
		return nil
	}

	// Fall back to stop + serve
	pid, _, _ := readPIDFile()
	if pid > 0 {
		fmt.Println("Stopping existing Dashboard...")
		if err := runDashboardStop(cmd, args); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return runDashboardServe(cmd, args)
}
