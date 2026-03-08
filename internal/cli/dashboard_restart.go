package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dashboardRestartCmd = &cobra.Command{
	Use:     "restart",
	Short:   "Restart the Dashboard server (stop then serve)",
	Example: "  clawsandbox dashboard restart\n  clawsandbox dashboard restart --port 9090",
	RunE:    runDashboardRestart,
}

func init() {
	dashboardRestartCmd.Flags().IntVar(&dashboardServePort, "port", 8080, "HTTP listen port")
	dashboardRestartCmd.Flags().StringVar(&dashboardServeHost, "host", "127.0.0.1", "HTTP listen host")
}

func runDashboardRestart(cmd *cobra.Command, args []string) error {
	pid, _, _ := readPIDFile()
	if pid > 0 {
		fmt.Println("Stopping existing Dashboard...")
		if err := runDashboardStop(cmd, args); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	return runDashboardServe(cmd, args)
}
