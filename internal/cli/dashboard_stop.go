package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
)

var dashboardStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop the running Dashboard server",
	Example: "  clawsandbox dashboard stop",
	RunE:    runDashboardStop,
}

func runDashboardStop(cmd *cobra.Command, args []string) error {
	pid, pidPath, err := readPIDFile()
	if err != nil {
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	fmt.Printf("Stopping Dashboard (pid %d) ... ", pid)
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidPath)
		return fmt.Errorf("failed to stop: %w", err)
	}

	for i := 0; i < 50; i++ {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	os.Remove(pidPath)
	fmt.Println("✓")
	return nil
}

func readPIDFile() (int, string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return 0, "", err
	}
	pidPath := filepath.Join(dir, "serve.pid")
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, "", fmt.Errorf("Dashboard is not running (no PID file at %s)", pidPath)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, pidPath, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, pidPath, nil
}
