//go:build linux

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/weiyong1024/clawfleet/internal/config"
)

// NewServiceManager returns systemd if available, otherwise falls back to PID-based.
func NewServiceManager() ServiceManager {
	if hasSystemdUser() {
		return &systemdManager{}
	}
	return &pidManager{}
}

func hasSystemdUser() bool {
	path, err := exec.LookPath("systemctl")
	if err != nil || path == "" {
		return false
	}
	out, err := exec.Command("systemctl", "--user", "is-system-running").CombinedOutput()
	if err != nil {
		s := strings.TrimSpace(string(out))
		return s == "running" || s == "degraded"
	}
	return true
}

// --- systemd implementation ---

const systemdServiceName = "clawfleet"

var unitTemplate = template.Must(template.New("unit").Parse(`[Unit]
Description=ClawFleet Dashboard
After=network.target docker.service

[Service]
Type=simple
ExecStart={{.BinaryPath}} dashboard serve --port {{.Port}} --host {{.Host}}
Restart=on-failure
RestartSec=5
StandardOutput=append:{{.LogPath}}
StandardError=append:{{.ErrPath}}

[Install]
WantedBy=default.target
`))

type systemdManager struct{}

func (m *systemdManager) unitPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "systemd", "user", systemdServiceName+".service")
}

func (m *systemdManager) Install(binaryPath string, port int, host string) error {
	logP := dashboardLogPath()
	errP := strings.TrimSuffix(logP, ".log") + ".err"

	data := struct {
		BinaryPath, Host, LogPath, ErrPath string
		Port                               int
	}{
		BinaryPath: binaryPath,
		Port:       port,
		Host:       host,
		LogPath:    logP,
		ErrPath:    errP,
	}

	path := m.unitPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating systemd user dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating unit file: %w", err)
	}
	defer f.Close()

	if err := unitTemplate.Execute(f, data); err != nil {
		return fmt.Errorf("writing unit file: %w", err)
	}

	if err := exec.Command("systemctl", "--user", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("daemon-reload: %w", err)
	}
	if err := exec.Command("systemctl", "--user", "enable", systemdServiceName).Run(); err != nil {
		return fmt.Errorf("enable service: %w", err)
	}
	return nil
}

func (m *systemdManager) Uninstall() error {
	_ = m.Stop()
	_ = exec.Command("systemctl", "--user", "disable", systemdServiceName).Run()
	path := m.unitPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing unit file: %w", err)
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (m *systemdManager) Start() error {
	return exec.Command("systemctl", "--user", "start", systemdServiceName).Run()
}

func (m *systemdManager) Stop() error {
	return exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
}

func (m *systemdManager) Restart() error {
	return exec.Command("systemctl", "--user", "restart", systemdServiceName).Run()
}

func (m *systemdManager) IsInstalled() (bool, error) {
	_, err := os.Stat(m.unitPath())
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

func (m *systemdManager) IsRunning() (bool, error) {
	out, err := exec.Command("systemctl", "--user", "is-active", systemdServiceName).Output()
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(string(out)) == "active", nil
}

func (m *systemdManager) Status() (DaemonStatus, error) {
	st := DaemonStatus{ServiceMgr: "systemd", LogPath: dashboardLogPath()}
	running, _ := m.IsRunning()
	st.Running = running
	if running {
		out, err := exec.Command("systemctl", "--user", "show", systemdServiceName, "--property=MainPID").Output()
		if err == nil {
			s := strings.TrimPrefix(strings.TrimSpace(string(out)), "MainPID=")
			if pid, err := strconv.Atoi(s); err == nil && pid > 0 {
				st.PID = pid
			}
		}
	}
	return st, nil
}

// --- PID-based fallback ---

type pidManager struct{}

func (m *pidManager) Install(binaryPath string, port int, host string) error {
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

	cmd := exec.Command(binPath, "dashboard", "serve")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

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
