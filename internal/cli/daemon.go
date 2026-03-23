package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/weiyong1024/clawfleet/internal/config"
)

// ServiceManager abstracts platform-specific daemon management.
type ServiceManager interface {
	// Install creates the service definition (systemd unit / launchd plist).
	Install(binaryPath string, port int, host string) error
	// Uninstall removes the service definition.
	Uninstall() error
	// Start starts the daemon via the platform service manager.
	Start() error
	// Stop stops the daemon.
	Stop() error
	// Restart restarts the daemon.
	Restart() error
	// IsInstalled reports whether the service definition exists.
	IsInstalled() (bool, error)
	// IsRunning reports whether the daemon is currently running.
	IsRunning() (bool, error)
	// Status returns daemon status info.
	Status() (DaemonStatus, error)
}

// DaemonStatus holds human-readable daemon state.
type DaemonStatus struct {
	Running    bool
	PID        int
	Port       int
	Host       string
	ServiceMgr string // "launchd", "systemd", or "pid"
	LogPath    string
}

// NewServiceManager returns the appropriate ServiceManager for the platform.
// Implemented in daemon_launchd.go (darwin), daemon_linux.go (linux),
// and daemon_other.go (other platforms).

// logDir returns the daemon log directory, creating it if necessary.
func logDir() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	p := filepath.Join(dir, "logs")
	if err := os.MkdirAll(p, 0755); err != nil {
		return "", fmt.Errorf("creating log dir: %w", err)
	}
	return p, nil
}

// dashboardLogPath returns the path to the dashboard log file.
func dashboardLogPath() string {
	d, err := logDir()
	if err != nil {
		return "~/.clawfleet/logs/dashboard.log"
	}
	return filepath.Join(d, "dashboard.log")
}

// waitForDashboard polls until the dashboard process is running or timeout.
func waitForDashboard(mgr ServiceManager, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		running, _ := mgr.IsRunning()
		if running {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

// pidFromFile reads the PID from the standard PID file.
func pidFromFile() (int, error) {
	dir, err := config.DataDir()
	if err != nil {
		return 0, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "serve.pid"))
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	// Verify alive
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, err
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, fmt.Errorf("process %d not alive", pid)
	}
	return pid, nil
}
