package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	docker "github.com/fsouza/go-dockerclient"
)

// ConfigureParams holds OpenClaw configuration parameters.
type ConfigureParams struct {
	ContainerID  string
	Provider     string // e.g. "anthropic", "openai"
	APIKey       string
	Model        string // e.g. "claude-sonnet-4-6"
	Channel      string // e.g. "telegram"
	ChannelToken string // bot token
}

// Configure runs openclaw CLI commands inside the container to set up the instance.
func Configure(cli *docker.Client, p ConfigureParams) error {
	// Stop the gateway if it is already running (reconfigure case).
	// This prevents config writes from triggering a hot-reload self-restart
	// that spawns orphan child processes supervisor cannot track (port conflict),
	// and avoids the gateway reloading with an incomplete intermediate config.
	_ = dockerExecAs(cli, p.ContainerID, "root", []string{
		"supervisorctl", "stop", "openclaw",
	})

	// Step 1: onboard with API key (runs as "node" — writes to ~node/.openclaw/)
	apiKeyFlag := fmt.Sprintf("--%s-api-key", p.Provider)
	if err := dockerExecAs(cli, p.ContainerID, "node", []string{
		"openclaw", "onboard",
		"--non-interactive", "--accept-risk", "--flow", "quickstart",
		apiKeyFlag, p.APIKey,
		"--skip-channels", "--skip-skills", "--skip-daemon", "--skip-ui",
		"--skip-health",
	}); err != nil {
		return fmt.Errorf("onboard: %w", err)
	}

	// Step 2: set default model (runs as "node")
	// OpenClaw expects fully qualified model IDs like "openai/gpt-4o".
	// If the user passes a bare model name, prefix it with the provider.
	if p.Model != "" {
		model := p.Model
		if !strings.Contains(model, "/") {
			model = p.Provider + "/" + model
		}
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "models", "set", model,
		}); err != nil {
			return fmt.Errorf("models set: %w", err)
		}
	}

	// Step 3: enable channel plugin if specified (must happen before gateway
	// starts so the plugin is loaded on boot)
	if p.Channel != "" {
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "plugins", "enable", p.Channel,
		}); err != nil {
			return fmt.Errorf("plugins enable %s: %w", p.Channel, err)
		}
	}

	// Step 4: start openclaw gateway via supervisord (runs as "root" — supervisord
	// socket is owned by root; the gateway process itself runs as "node" per
	// the [program:openclaw] user=node directive)
	if err := dockerExecAs(cli, p.ContainerID, "root", []string{
		"supervisorctl", "start", "openclaw",
	}); err != nil {
		return fmt.Errorf("supervisorctl start: %w", err)
	}

	// Step 5: wait for gateway to be ready
	if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
		return fmt.Errorf("waiting for gateway: %w", err)
	}

	// Step 6: add channel account (requires running gateway with plugin loaded)
	if p.Channel != "" && p.ChannelToken != "" {
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "channels", "add",
			"--channel", p.Channel,
			"--token", p.ChannelToken,
		}); err != nil {
			return fmt.Errorf("channels add: %w", err)
		}

		// Step 7: stop gateway before writing policy changes so they are
		// applied offline — no hot-reload with incomplete intermediate config.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "stop", "openclaw",
		}); err != nil {
			return fmt.Errorf("supervisorctl stop before policies: %w", err)
		}

		// Set DM and group policies to "open" so the bot responds
		// without pairing. allowFrom must include "*" when policy is "open".
		channelCfg := fmt.Sprintf("channels.%s", p.Channel)
		policySteps := []struct{ path, value string }{
			{channelCfg + ".allowFrom", `["*"]`},
			{channelCfg + ".dmPolicy", "open"},
			{channelCfg + ".groupAllowFrom", `["*"]`},
			{channelCfg + ".groupPolicy", "open"},
		}
		for _, s := range policySteps {
			args := []string{"openclaw", "config", "set", s.path, s.value}
			// Arrays need --strict-json to be parsed correctly.
			if strings.HasPrefix(s.value, "[") {
				args = append(args, "--strict-json")
			}
			if err := dockerExecAs(cli, p.ContainerID, "node", args); err != nil {
				return fmt.Errorf("config set %s: %w", s.path, err)
			}
		}

		// Step 8: start gateway with the complete, final config.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw",
		}); err != nil {
			return fmt.Errorf("supervisorctl start after policies: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for gateway restart: %w", err)
		}
	}

	return nil
}

// ConfigInfo holds the configuration status of an instance.
type ConfigInfo struct {
	Configured       bool   `json:"configured"`
	Provider         string `json:"provider,omitempty"`
	Model            string `json:"model,omitempty"`
	Channel          string `json:"channel,omitempty"`
	APIKeyHint       string `json:"api_key_hint,omitempty"`
	ChannelTokenHint string `json:"channel_token_hint,omitempty"`
}

// maskLast4 returns "••••xxxx" where xxxx is the last 4 characters.
func maskLast4(s string) string {
	if len(s) <= 4 {
		return ""
	}
	return "••••" + s[len(s)-4:]
}

// ConfigStatus checks if the instance is configured by reading the config file.
func ConfigStatus(cli *docker.Client, containerID string) (*ConfigInfo, error) {
	out, err := dockerExecOutputAs(cli, containerID, "node", []string{
		"cat", "/home/node/.openclaw/openclaw.json",
	})
	if err != nil {
		return &ConfigInfo{Configured: false}, nil
	}

	// Parse the main config JSON.
	var cfg struct {
		Agents struct {
			Defaults struct {
				Model struct {
					Primary string `json:"primary"`
				} `json:"model"`
			} `json:"defaults"`
		} `json:"agents"`
		Channels map[string]struct {
			BotToken string `json:"botToken"`
			Token    string `json:"token"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return &ConfigInfo{Configured: true}, nil
	}

	info := &ConfigInfo{Configured: true}

	// Extract model and provider from "openai/gpt-4o" format.
	if m := cfg.Agents.Defaults.Model.Primary; m != "" {
		info.Model = m
		if parts := strings.SplitN(m, "/", 2); len(parts) == 2 {
			info.Provider = parts[0]
		}
	}

	// Read API key hint from auth-profiles.json.
	authOut, err := dockerExecOutputAs(cli, containerID, "node", []string{
		"cat", "/home/node/.openclaw/agents/main/agent/auth-profiles.json",
	})
	if err == nil {
		var authCfg struct {
			Profiles map[string]struct {
				Key string `json:"key"`
			} `json:"profiles"`
		}
		if json.Unmarshal([]byte(authOut), &authCfg) == nil {
			for _, p := range authCfg.Profiles {
				if p.Key != "" {
					info.APIKeyHint = maskLast4(p.Key)
					break
				}
			}
		}
	}

	// Find the first channel and its token hint.
	for name, ch := range cfg.Channels {
		info.Channel = name
		token := ch.BotToken
		if token == "" {
			token = ch.Token
		}
		if token != "" {
			info.ChannelTokenHint = maskLast4(token)
		}
		break
	}

	return info, nil
}

// dockerExecAs runs a command inside a container as the specified user.
func dockerExecAs(cli *docker.Client, containerID, user string, cmd []string) error {
	exec, err := cli.CreateExec(docker.CreateExecOptions{
		Container:    containerID,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		User:         user,
	})
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := cli.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: &stdout,
		ErrorStream:  &stderr,
	}); err != nil {
		return fmt.Errorf("start exec: %w", err)
	}

	inspect, err := cli.InspectExec(exec.ID)
	if err != nil {
		return fmt.Errorf("inspect exec: %w", err)
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("exit code %d: %s", inspect.ExitCode, strings.TrimSpace(stderr.String()))
	}

	return nil
}

// dockerExecOutputAs runs a command as the specified user and returns its stdout.
func dockerExecOutputAs(cli *docker.Client, containerID, user string, cmd []string) (string, error) {
	exec, err := cli.CreateExec(docker.CreateExecOptions{
		Container:    containerID,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		User:         user,
	})
	if err != nil {
		return "", fmt.Errorf("create exec: %w", err)
	}

	var stdout, stderr bytes.Buffer
	if err := cli.StartExec(exec.ID, docker.StartExecOptions{
		OutputStream: &stdout,
		ErrorStream:  &stderr,
	}); err != nil {
		return "", fmt.Errorf("start exec: %w", err)
	}

	inspect, err := cli.InspectExec(exec.ID)
	if err != nil {
		return "", fmt.Errorf("inspect exec: %w", err)
	}
	if inspect.ExitCode != 0 {
		return "", fmt.Errorf("exit code %d: %s", inspect.ExitCode, strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}

// waitForGateway polls the gateway health endpoint until it responds or timeout.
func waitForGateway(cli *docker.Client, containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_, err := dockerExecOutputAs(cli, containerID, "node", []string{
			"curl", "-sf", "http://127.0.0.1:18789/health",
		})
		if err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("gateway did not become ready within %s", timeout)
}
