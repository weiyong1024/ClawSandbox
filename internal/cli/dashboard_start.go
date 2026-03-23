package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var dashboardStartPort int
var dashboardStartHost string

var dashboardStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Dashboard as a background daemon",
	Long: `Start the Dashboard in the background with auto-restart.
Uses launchd on macOS and systemd on Linux.
The daemon will restart automatically if it crashes.`,
	Example: "  clawfleet dashboard start\n  clawfleet dashboard start --port 9090",
	RunE:    runDashboardStart,
}

func init() {
	dashboardStartCmd.Flags().IntVar(&dashboardStartPort, "port", 8080, "HTTP listen port")
	dashboardStartCmd.Flags().StringVar(&dashboardStartHost, "host", "127.0.0.1", "HTTP listen host")
}

func runDashboardStart(cmd *cobra.Command, args []string) error {
	mgr := NewServiceManager()

	// Check if already running
	running, _ := mgr.IsRunning()
	if running {
		st, _ := mgr.Status()
		fmt.Printf("Dashboard is already running (pid %d).\n", st.PID)
		fmt.Printf("URL: http://%s:%d\n", dashboardStartHost, dashboardStartPort)
		return nil
	}

	// Find binary path
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding binary path: %w", err)
	}

	// Install service if not already installed
	installed, _ := mgr.IsInstalled()
	if !installed {
		fmt.Printf("Installing Dashboard service ... ")
		if err := mgr.Install(binPath, dashboardStartPort, dashboardStartHost); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		fmt.Println("done")
	} else {
		// Re-install to update binary path and port
		if err := mgr.Install(binPath, dashboardStartPort, dashboardStartHost); err != nil {
			return fmt.Errorf("updating service: %w", err)
		}
	}

	// Start the service
	fmt.Printf("Starting Dashboard ... ")
	if err := mgr.Start(); err != nil {
		return fmt.Errorf("start failed: %w", err)
	}

	// Wait for it to be running
	if !waitForDashboard(mgr, 5*time.Second) {
		fmt.Println("timeout")
		fmt.Printf("Dashboard may still be starting. Check logs at: %s\n", dashboardLogPath())
		return fmt.Errorf("dashboard did not start within 5 seconds")
	}

	fmt.Println("done")
	fmt.Printf("Dashboard: http://%s:%d\n", dashboardStartHost, dashboardStartPort)
	fmt.Printf("Logs:      %s\n", dashboardLogPath())
	return nil
}
