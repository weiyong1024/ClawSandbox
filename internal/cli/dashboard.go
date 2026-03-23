package cli

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Web Dashboard commands",
	Long:  `Manage the ClawFleet Web Dashboard — start the server or open it in your browser.`,
}

var dashboardOpenPort int

var dashboardOpenCmd = &cobra.Command{
	Use:     "open",
	Short:   "Open the Dashboard in your browser",
	Example: "  clawfleet dashboard open\n  clawfleet dashboard open --port 9090",
	RunE: func(cmd *cobra.Command, args []string) error {
		url := fmt.Sprintf("http://localhost:%d", dashboardOpenPort)
		fmt.Printf("Opening Dashboard at %s\n", url)
		return openURL(url)
	},
}

func init() {
	dashboardOpenCmd.Flags().IntVar(&dashboardOpenPort, "port", 8080, "Dashboard port")
	dashboardCmd.AddCommand(dashboardServeCmd, dashboardStartCmd, dashboardStopCmd, dashboardRestartCmd, dashboardStatusCmd, dashboardOpenCmd)
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		fmt.Printf("Open this URL in your browser: %s\n", url)
		return nil
	}
	return cmd.Start()
}
