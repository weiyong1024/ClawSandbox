//go:build !darwin && !linux

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/clawfleet/clawfleet/internal/config"
)

// NewServiceManager returns a PID-based ServiceManager on unsupported platforms.
func NewServiceManager() ServiceManager {
	return &pidManager{}
}

type pidManager struct {
	port int
	host string
}

func (m *pidManager) Install(binaryPath string, port int, host string) error {
	m.port = port
	m.host = host
	return nil
}

func (m *pidManager) Uninstall() error {
	return m.Stop()
}

func (m *pidManager) Start() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding binary: %w", err)
	}

	logP := dashboardLogPath()
	logFile, err := os.OpenFile(logP, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	args := []string{"dashboard", "serve"}
	if m.host != "" {
		args = append(args, "--host", m.host)
	}
	if m.port > 0 {
		args = append(args, "--port", fmt.Sprintf("%d", m.port))
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting dashboard: %w", err)
	}
	logFile.Close()
	return nil
}

func (m *pidManager) Stop() error {
	pid, err := pidFromFile()
	if err != nil {
		return fmt.Errorf("Dashboard is not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to stop: %w", err)
	}
	for i := 0; i < 50; i++ {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	dir, err := config.DataDir()
	if err == nil {
		os.Remove(filepath.Join(dir, "serve.pid"))
	}
	return nil
}

func (m *pidManager) Restart() error {
	_ = m.Stop()
	return m.Start()
}

func (m *pidManager) IsInstalled() (bool, error) {
	return true, nil
}

func (m *pidManager) IsRunning() (bool, error) {
	_, err := pidFromFile()
	return err == nil, nil
}

func (m *pidManager) Status() (DaemonStatus, error) {
	st := DaemonStatus{ServiceMgr: "pid", LogPath: dashboardLogPath()}
	pid, err := pidFromFile()
	if err == nil {
		st.Running = true
		st.PID = pid
	}
	return st, nil
}
