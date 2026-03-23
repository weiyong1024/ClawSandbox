package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var dashboardStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show Dashboard daemon status",
	Example: "  clawfleet dashboard status",
	RunE:    runDashboardStatus,
}

func runDashboardStatus(cmd *cobra.Command, args []string) error {
	mgr := NewServiceManager()

	st, err := mgr.Status()
	if err != nil {
		return fmt.Errorf("getting status: %w", err)
	}

	fmt.Println("Dashboard Status")
	if st.Running {
		fmt.Println("  State:    running")
		if st.PID > 0 {
			fmt.Printf("  PID:      %d\n", st.PID)
		}
		fmt.Printf("  Manager:  %s\n", st.ServiceMgr)
		fmt.Printf("  Log:      %s\n", st.LogPath)
	} else {
		fmt.Println("  State:    stopped")
		fmt.Printf("  Manager:  %s\n", st.ServiceMgr)
		fmt.Println()
		fmt.Println("  Run 'clawfleet dashboard start' to start the daemon.")
	}

	return nil
}
