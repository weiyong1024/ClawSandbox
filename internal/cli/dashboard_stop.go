package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawfleet/internal/config"
)

var dashboardStopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop the running Dashboard server",
	Example: "  clawfleet dashboard stop",
	RunE:    runDashboardStop,
}

func runDashboardStop(cmd *cobra.Command, args []string) error {
	// Try service manager first
	mgr := NewServiceManager()
	if installed, _ := mgr.IsInstalled(); installed {
		if running, _ := mgr.IsRunning(); running {
			fmt.Printf("Stopping Dashboard daemon ... ")
			if err := mgr.Stop(); err != nil {
				return fmt.Errorf("failed to stop: %w", err)
			}
			fmt.Println("done")
			return nil
		}
	}

	// Fall back to PID-based stop
	pid, pidPath, err := readPIDFile()
	if err != nil {
		// PID file missing — try to find the process by port.
		pid, err = findPIDByPort(dashboardServePort)
		if err != nil {
			return fmt.Errorf("Dashboard is not running")
		}
		pidPath = ""
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}

	// Verify the process is actually alive.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		if pidPath != "" {
			os.Remove(pidPath)
		}
		return fmt.Errorf("Dashboard is not running (stale PID %d)", pid)
	}

	fmt.Printf("Stopping Dashboard (pid %d) ... ", pid)
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		if pidPath != "" {
			os.Remove(pidPath)
		}
		return fmt.Errorf("failed to stop: %w", err)
	}

	for i := 0; i < 50; i++ {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if pidPath != "" {
		os.Remove(pidPath)
	}
	fmt.Println("done")
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
		return 0, "", fmt.Errorf("no PID file")
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, pidPath, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, pidPath, nil
}

// findPIDByPort uses lsof to find the PID of the process listening on the given port.
func findPIDByPort(port int) (int, error) {
	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port), "-sTCP:LISTEN").Output()
	if err != nil {
		return 0, fmt.Errorf("no process found on port %d", port)
	}
	line := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	pid, err := strconv.Atoi(line)
	if err != nil {
		return 0, fmt.Errorf("unexpected lsof output: %s", line)
	}
	return pid, nil
}
