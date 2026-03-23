//go:build darwin

package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// NewServiceManager returns a launchd-based ServiceManager on macOS.
func NewServiceManager() ServiceManager {
	return &launchdManager{}
}

const launchdLabel = "io.clawfleet.dashboard"

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>io.clawfleet.dashboard</string>
  <key>ProgramArguments</key>
  <array>
    <string>{{.BinaryPath}}</string>
    <string>dashboard</string>
    <string>serve</string>
    <string>--port</string>
    <string>{{.Port}}</string>
    <string>--host</string>
    <string>{{.Host}}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <true/>
  <key>StandardOutPath</key>
  <string>{{.LogPath}}</string>
  <key>StandardErrorPath</key>
  <string>{{.ErrPath}}</string>
</dict>
</plist>
`))

type launchdManager struct{}

func (m *launchdManager) plistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func (m *launchdManager) Install(binaryPath string, port int, host string) error {
	logP := dashboardLogPath()
	errP := strings.TrimSuffix(logP, ".log") + ".err"

	data := struct {
		BinaryPath, Host, LogPath, ErrPath string
		Port                               string
	}{
		BinaryPath: binaryPath,
		Port:       fmt.Sprintf("%d", port),
		Host:       host,
		LogPath:    logP,
		ErrPath:    errP,
	}

	path := m.plistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating LaunchAgents dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating plist: %w", err)
	}
	defer f.Close()

	if err := plistTemplate.Execute(f, data); err != nil {
		return fmt.Errorf("writing plist: %w", err)
	}
	return nil
}

func (m *launchdManager) Uninstall() error {
	_ = m.Stop()
	path := m.plistPath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing plist: %w", err)
	}
	return nil
}

func (m *launchdManager) Start() error {
	return exec.Command("launchctl", "load", m.plistPath()).Run()
}

func (m *launchdManager) Stop() error {
	return exec.Command("launchctl", "unload", m.plistPath()).Run()
}

func (m *launchdManager) Restart() error {
	_ = m.Stop()
	return m.Start()
}

func (m *launchdManager) IsInstalled() (bool, error) {
	_, err := os.Stat(m.plistPath())
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *launchdManager) IsRunning() (bool, error) {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(out), launchdLabel), nil
}

func (m *launchdManager) Status() (DaemonStatus, error) {
	st := DaemonStatus{ServiceMgr: "launchd", LogPath: dashboardLogPath()}

	running, _ := m.IsRunning()
	st.Running = running

	if running {
		if pid, err := pidFromFile(); err == nil {
			st.PID = pid
		}
	}

	return st, nil
}
