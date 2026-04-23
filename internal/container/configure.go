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
	ContainerID     string
	Provider        string // e.g. "anthropic", "openai", "openai-codex"
	APIKey          string
	Model           string      // e.g. "claude-sonnet-4-6"
	Channel         string      // e.g. "telegram", "lark"
	ChannelToken    string      // bot token (Telegram, Discord, Slack)
	ChannelAppToken string      // Slack app token for Socket Mode
	AppID           string      // Lark/Feishu App ID
	AppSecret       string      // Lark/Feishu App Secret
	BotName         string      // bot display name for text @mention detection
	Soul            *SoulParams // optional character to inject before gateway starts
	// OAuth fields (openai-codex)
	OAuthRefresh   string // Codex refresh token
	OAuthExpires   int64  // Codex token expiry (ms)
	OAuthAccountID string // Codex ChatGPT account ID
}

type configSetStep struct {
	path       string
	value      string
	strictJSON bool
}

// openclawChannelName maps ClawFleet channel names to OpenClaw plugin IDs.
// OpenClaw uses "feishu" as the plugin/channel name, but ClawFleet presents
// it as "lark" in the UI for international users.
func openclawChannelName(channel string) string {
	if channel == "lark" {
		return "feishu"
	}
	return channel
}

func applyConfigSteps(cli *docker.Client, containerID, user string, steps []configSetStep) error {
	for _, step := range steps {
		args := []string{"openclaw", "config", "set", step.path, step.value}
		if step.strictJSON {
			args = append(args, "--strict-json")
		}
		if err := dockerExecAs(cli, containerID, user, args); err != nil {
			return fmt.Errorf("config set %s: %w", step.path, err)
		}
	}
	return nil
}

func channelPolicySteps(channel, channelCfg string) []configSetStep {
	steps := []configSetStep{
		{path: channelCfg + ".allowFrom", value: `["*"]`, strictJSON: true},
		{path: channelCfg + ".dmPolicy", value: "open"},
		{path: channelCfg + ".groupPolicy", value: "open"},
	}

	switch channel {
	case "lark":
		steps = append(steps, configSetStep{
			path:  channelCfg + ".allowBots",
			value: "mentions",
		})
	case "slack":
		// OpenClaw's Slack schema only accepts a boolean here.
		steps = append(steps, configSetStep{
			path:       channelCfg + ".allowBots",
			value:      "true",
			strictJSON: true,
		})
	case "telegram":
		steps = append(steps, configSetStep{
			path:       channelCfg + ".groupAllowFrom",
			value:      `["*"]`,
			strictJSON: true,
		})
	case "discord":
		steps = append(steps, configSetStep{
			path:  channelCfg + ".allowBots",
			value: "mentions",
		})
	}

	return steps
}

// Configure runs openclaw CLI commands inside the container to set up the instance.
func Configure(cli *docker.Client, p ConfigureParams) error {
	// Stop the gateway and its LAN bridge if already running (reconfigure case).
	_ = dockerExecAs(cli, p.ContainerID, "root", []string{
		"supervisorctl", "stop", "openclaw", "gateway-bridge",
	})

	// Onboard: create workspace and set up auth credentials.
	// Codex OAuth uses --auth-choice skip + direct auth-profiles.json injection,
	// since the interactive OAuth flow runs on the Dashboard host, not inside the container.
	if p.Provider == "openai-codex" {
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "onboard",
			"--non-interactive", "--accept-risk", "--flow", "quickstart",
			"--auth-choice", "skip",
			"--skip-channels", "--skip-skills", "--skip-daemon", "--skip-ui",
			"--skip-health",
		}); err != nil {
			return fmt.Errorf("onboard: %w", err)
		}
		if err := injectCodexAuthProfile(cli, p.ContainerID, p.APIKey, p.OAuthRefresh, p.OAuthExpires, p.OAuthAccountID); err != nil {
			return fmt.Errorf("inject codex auth: %w", err)
		}
	} else {
		// Map ClawFleet provider names to OpenClaw onboard flag names.
		// OpenClaw uses "gemini" not "google" for the API key flag.
		flagProvider := p.Provider
		if flagProvider == "google" {
			flagProvider = "gemini"
		}
		apiKeyFlag := fmt.Sprintf("--%s-api-key", flagProvider)
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "onboard",
			"--non-interactive", "--accept-risk", "--flow", "quickstart",
			apiKeyFlag, p.APIKey,
			"--skip-channels", "--skip-skills", "--skip-daemon", "--skip-ui",
			"--skip-health",
		}); err != nil {
			return fmt.Errorf("onboard: %w", err)
		}
	}

	// Inject SOUL.md immediately after onboard (which creates the workspace).
	// This must happen BEFORE the gateway starts so the character is part of
	// the initial system prompt bootstrap.
	if p.Soul != nil {
		if err := InjectSoul(cli, p.ContainerID, *p.Soul); err != nil {
			return fmt.Errorf("inject soul: %w", err)
		}
	}

	// Allow the Dashboard console proxy to access the Gateway web UI:
	// - auth.mode=none: loopback mode allows no-auth; Dashboard is the security layer.
	//   OpenClaw onboard auto-generates a token, so we must explicitly override to none.
	// - allowedOrigins=["*"]: permit WebSocket from any Dashboard host.
	if err := applyConfigSteps(cli, p.ContainerID, "node", []configSetStep{
		{path: "gateway.auth", value: `{"mode":"none"}`, strictJSON: true},
		{path: "gateway.controlUi.allowedOrigins", value: `["*"]`, strictJSON: true},
	}); err != nil {
		return fmt.Errorf("configure gateway access: %w", err)
	}

	// Set default model (runs as "node").
	// OpenClaw expects fully qualified model IDs like "openai/gpt-5.4".
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
	// starts so the plugin is loaded on boot).
	// Map ClawFleet channel names to OpenClaw plugin IDs (e.g. "lark" → "feishu").
	pluginName := openclawChannelName(p.Channel)
	if p.Channel != "" {
		// Feishu plugin requires npm dependencies that may not be installed
		// in older images. Install them if missing (idempotent, fast if present).
		if pluginName == "feishu" {
			_ = dockerExecAs(cli, p.ContainerID, "root", []string{
				"bash", "-c",
				"cd /usr/local/lib/node_modules/openclaw/extensions/feishu && npm install --omit=dev",
			})
		}
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "plugins", "enable", pluginName,
		}); err != nil {
			return fmt.Errorf("plugins enable %s: %w", pluginName, err)
		}
	}

	// Step 4: set up channel credentials and policies.
	//
	// Feishu/Lark uses config set (appId + appSecret) instead of channels add.
	// Its credentials and policies are written BEFORE the gateway starts to
	// avoid hot-reload race conditions.
	//
	// Slack Socket Mode writes both tokens offline via channels add.
	//
	// Telegram and Discord require a running gateway for
	// "channels add --token", so they follow the start→add→stop→policies→restart
	// pattern.
	if p.Channel != "" {
		switch p.Channel {
		case "lark":
			if p.AppID == "" || p.AppSecret == "" {
				return fmt.Errorf("Lark App ID and App Secret are required")
			}
		case "slack":
			if p.ChannelToken == "" {
				return fmt.Errorf("Slack bot token is required")
			}
			if p.ChannelAppToken == "" {
				return fmt.Errorf("Slack app token is required")
			}
		default:
			if p.ChannelToken == "" {
				return fmt.Errorf("channel token is required for %s", p.Channel)
			}
		}
	}

	if p.Channel == "lark" && p.AppID != "" && p.AppSecret != "" {
		// Feishu: write all config offline (no running gateway needed).
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "config", "set", "channels.feishu.appId", p.AppID,
		}); err != nil {
			return fmt.Errorf("config set channels.feishu.appId: %w", err)
		}
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "config", "set", "channels.feishu.appSecret", p.AppSecret,
		}); err != nil {
			return fmt.Errorf("config set channels.feishu.appSecret: %w", err)
		}
		channelCfg := "channels.feishu"
		if err := applyConfigSteps(cli, p.ContainerID, "node", channelPolicySteps("lark", channelCfg)); err != nil {
			return err
		}
		// Start gateway with the complete config.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw", "gateway-bridge",
		}); err != nil {
			return fmt.Errorf("supervisorctl start: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for gateway: %w", err)
		}
	} else if p.Channel == "slack" && p.ChannelToken != "" && p.ChannelAppToken != "" {
		if err := dockerExecAs(cli, p.ContainerID, "node", []string{
			"openclaw", "channels", "add",
			"--channel", "slack",
			"--bot-token", p.ChannelToken,
			"--app-token", p.ChannelAppToken,
		}); err != nil {
			return fmt.Errorf("channels add slack: %w", err)
		}

		channelCfg := "channels.slack"
		if err := applyConfigSteps(cli, p.ContainerID, "node", channelPolicySteps("slack", channelCfg)); err != nil {
			return err
		}

		if p.BotName != "" {
			agentsList := fmt.Sprintf(`[{"id":"main","identity":{"name":"%s"}}]`, p.BotName)
			if err := dockerExecAs(cli, p.ContainerID, "node", []string{
				"openclaw", "config", "set", "agents.list", agentsList, "--strict-json",
			}); err != nil {
				return fmt.Errorf("config set agents.list: %w", err)
			}
		}

		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw", "gateway-bridge",
		}); err != nil {
			return fmt.Errorf("supervisorctl start after Slack configure: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for Slack gateway start: %w", err)
		}
	} else if p.Channel != "" && p.ChannelToken != "" {
		// Telegram/Discord: start gateway → channels add → policies (no stop/restart).
		// channels add MUST run with gateway online so the Discord client initializes
		// properly. Policies are applied via hot-reload — no gateway restart needed.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw", "gateway-bridge",
		}); err != nil {
			return fmt.Errorf("supervisorctl start: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for gateway: %w", err)
		}

		// channels add with retry — ConfigMutationConflictError can occur when
		// the gateway's hot-reload races with the config write.
		var addErr error
		for attempt := 0; attempt < 3; attempt++ {
			addErr = dockerExecAs(cli, p.ContainerID, "node", []string{
				"openclaw", "channels", "add",
				"--channel", pluginName,
				"--token", p.ChannelToken,
			})
			if addErr == nil {
				break
			}
			time.Sleep(2 * time.Second)
		}
		if addErr != nil {
			return fmt.Errorf("channels add: %w", addErr)
		}

		// Stop gateway before writing policies. Multiple rapid config set calls
		// while gateway is running trigger cascading hot-reloads → restart loop.
		// Writing all policies offline then starting once avoids this entirely.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "stop", "openclaw",
		}); err != nil {
			return fmt.Errorf("supervisorctl stop before policies: %w", err)
		}

		channelCfg := fmt.Sprintf("channels.%s", pluginName)
		if err := applyConfigSteps(cli, p.ContainerID, "node", channelPolicySteps(p.Channel, channelCfg)); err != nil {
			return err
		}

		// Set agent identity name for text @mention detection.
		if p.BotName != "" {
			agentsList := fmt.Sprintf(`[{"id":"main","identity":{"name":"%s"}}]`, p.BotName)
			if err := dockerExecAs(cli, p.ContainerID, "node", []string{
				"openclaw", "config", "set", "agents.list", agentsList, "--strict-json",
			}); err != nil {
				return fmt.Errorf("config set agents.list: %w", err)
			}
		}

		// Final start with complete config. channels add already initialized
		// the Discord client state; this restart picks up all policies cleanly.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw", "gateway-bridge",
		}); err != nil {
			return fmt.Errorf("supervisorctl start after policies: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for gateway: %w", err)
		}
	} else if p.Channel == "" {
		// No channel — just start the gateway with model-only config.
		if err := dockerExecAs(cli, p.ContainerID, "root", []string{
			"supervisorctl", "start", "openclaw", "gateway-bridge",
		}); err != nil {
			return fmt.Errorf("supervisorctl start: %w", err)
		}
		if err := waitForGateway(cli, p.ContainerID, 30*time.Second); err != nil {
			return fmt.Errorf("waiting for gateway: %w", err)
		}
	}

	// Write .configured marker so gateway auto-starts on container restart.
	if err := dockerExecAs(cli, p.ContainerID, "node", []string{
		"touch", "/home/node/.openclaw/.configured",
	}); err != nil {
		return fmt.Errorf("writing .configured marker: %w", err)
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
			AppID    string `json:"appId"`
		} `json:"channels"`
	}
	if err := json.Unmarshal([]byte(out), &cfg); err != nil {
		return &ConfigInfo{Configured: true}, nil
	}

	info := &ConfigInfo{Configured: true}

	// Extract model and provider from "openai/gpt-5.4" format.
	if m := cfg.Agents.Defaults.Model.Primary; m != "" {
		info.Model = m
		if parts := strings.SplitN(m, "/", 2); len(parts) == 2 {
			info.Provider = parts[0]
		}
	}

	// Read API key hint from auth-profiles.json.
	// Supports both api_key profiles (key field) and OAuth profiles (access field).
	authOut, err := dockerExecOutputAs(cli, containerID, "node", []string{
		"cat", "/home/node/.openclaw/agents/main/agent/auth-profiles.json",
	})
	if err == nil {
		var authCfg struct {
			Profiles map[string]struct {
				Key    string `json:"key"`
				Access string `json:"access"`
			} `json:"profiles"`
		}
		if json.Unmarshal([]byte(authOut), &authCfg) == nil {
			for _, p := range authCfg.Profiles {
				if p.Key != "" {
					info.APIKeyHint = maskLast4(p.Key)
					break
				}
				if p.Access != "" {
					info.APIKeyHint = "OAuth ✓"
					break
				}
			}
		}
	}

	// Find the first channel and its token/credential hint.
	for name, ch := range cfg.Channels {
		info.Channel = name
		token := ch.BotToken
		if token == "" {
			token = ch.Token
		}
		if token == "" {
			token = ch.AppID // Feishu uses appId instead of token
		}
		if token != "" {
			info.ChannelTokenHint = maskLast4(token)
		}
		break
	}

	return info, nil
}

// Teammate describes another bot in the fleet, for roster injection into SOUL.md.
type Teammate struct {
	Name    string // Character.Name — the @mention key
	Bio     string // Full Character.Bio
	Channel string // "discord", "telegram", etc.
}

// SoulParams holds the fields for rendering a SOUL.md character file.
type SoulParams struct {
	Name       string
	Bio        string
	Lore       string
	Style      string
	Topics     string
	Adjectives string
	Teammates  []Teammate
}

// RenderSoulMarkdown produces the SOUL.md content from the given parameters.
// Exported as a pure function for testability.
func RenderSoulMarkdown(p SoulParams) string {
	var sb strings.Builder
	sb.WriteString("# " + p.Name + "\n")
	sb.WriteString("\n**You are " + p.Name + ". Stay in character at all times. Every response must reflect this persona's voice, personality, and perspective. Never break character or revert to a generic assistant.**\n")
	if p.Bio != "" {
		sb.WriteString("\n## Bio\n" + p.Bio + "\n")
	}
	if p.Lore != "" {
		sb.WriteString("\n## Background\n" + p.Lore + "\n")
	}
	if p.Style != "" {
		sb.WriteString("\n## Communication Style\n" + p.Style + "\n")
	}
	if p.Topics != "" {
		sb.WriteString("\n## Topics of Interest\n" + p.Topics + "\n")
	}
	if p.Adjectives != "" {
		sb.WriteString("\n## Personality Traits\n" + p.Adjectives + "\n")
	}

	if len(p.Teammates) > 0 {
		sb.WriteString("\n## Your Team\n")
		sb.WriteString("You are part of a team of AI agents in a group conversation. Here are your teammates:\n\n")
		for _, t := range p.Teammates {
			sb.WriteString(fmt.Sprintf("- **%s**", t.Name))
			if t.Bio != "" {
				sb.WriteString(": " + t.Bio)
			}
			if t.Channel != "" {
				sb.WriteString(" (" + t.Channel + ")")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n### How to collaborate\n")
		sb.WriteString("- When a user's message touches a teammate's expertise, @mention them by name (e.g. @" + p.Teammates[0].Name + ") to invite them into the discussion.\n")
		sb.WriteString("- When you are @mentioned by a teammate, contribute your perspective on the topic.\n")
		sb.WriteString("- Do NOT @mention a teammate who has already spoken on this topic — build on their point instead.\n")
		sb.WriteString("- After the key perspectives have been shared, wrap up with a brief synthesis and yield the floor to the human.\n")
		sb.WriteString("- When a human speaks, they have priority — respond to them directly.\n")
	}

	return sb.String()
}

// InjectSoul renders the character fields into a SOUL.md file and writes it
// into the container. The OpenClaw gateway watches this file for changes,
// so no restart is needed.
func InjectSoul(cli *docker.Client, containerID string, p SoulParams) error {
	content := RenderSoulMarkdown(p)
	// Write SOUL.md to the workspace directory where OpenClaw actually reads it.
	// The workspace is at ~/.openclaw/workspace/ and Gateway watches it for changes.
	return dockerExecAs(cli, containerID, "node", []string{
		"bash", "-c", fmt.Sprintf("cat > /home/node/.openclaw/workspace/SOUL.md << 'CLAWFLEET_EOF'\n%sCLAWFLEET_EOF", content),
	})
}

// ExecAs runs a command inside a container as the specified user (public wrapper).
func ExecAs(cli *docker.Client, containerID, user string, cmd []string) error {
	return dockerExecAs(cli, containerID, user, cmd)
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

// injectCodexAuthProfile writes the Codex OAuth credentials into the container's
// auth-profiles.json. It reads the existing file (to preserve other providers),
// adds/updates the openai-codex:default profile, and writes it back.
func injectCodexAuthProfile(cli *docker.Client, containerID, accessToken, refreshToken string, expires int64, accountID string) error {
	const authDir = "/home/node/.openclaw/agents/main/agent"
	const authPath = authDir + "/auth-profiles.json"

	// Ensure the directory exists — onboard --auth-choice skip may not create it.
	if err := dockerExecAs(cli, containerID, "node", []string{"mkdir", "-p", authDir}); err != nil {
		return fmt.Errorf("mkdir auth dir: %w", err)
	}

	// Read existing auth profiles (may not exist yet on first configure).
	existing, _ := dockerExecOutputAs(cli, containerID, "node", []string{"cat", authPath})

	var store map[string]any
	if existing != "" {
		if err := json.Unmarshal([]byte(existing), &store); err != nil {
			store = nil
		}
	}
	if store == nil {
		store = map[string]any{"version": float64(1), "profiles": map[string]any{}}
	}

	profiles, ok := store["profiles"].(map[string]any)
	if !ok {
		profiles = map[string]any{}
		store["profiles"] = profiles
	}

	profiles["openai-codex:default"] = map[string]any{
		"type":      "oauth",
		"provider":  "openai-codex",
		"access":    accessToken,
		"refresh":   refreshToken,
		"expires":   expires,
		"accountId": accountID,
	}

	// Update lastGood so OpenClaw uses this profile by default.
	lastGood, ok := store["lastGood"].(map[string]any)
	if !ok {
		lastGood = map[string]any{}
		store["lastGood"] = lastGood
	}
	lastGood["openai-codex"] = "openai-codex:default"

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal auth profiles: %w", err)
	}

	// Write the file via docker exec.
	return dockerExecAs(cli, containerID, "node", []string{
		"bash", "-c", fmt.Sprintf("cat > %s << 'CLAWFLEET_EOF'\n%s\nCLAWFLEET_EOF", authPath, string(data)),
	})
}

// HermesConfigureParams holds Hermes-specific configuration parameters.
type HermesConfigureParams struct {
	ContainerID  string
	Provider     string // e.g. "openai", "openai-codex", "anthropic"
	APIKey       string // API key or OAuth access token
	Model        string // e.g. "gpt-5.4-mini"
	Channel      string // e.g. "discord", "telegram"
	ChannelToken string // bot token
	// OAuth fields (openai-codex)
	OAuthRefresh   string
	OAuthExpires   int64
	OAuthAccountID string
}

// ConfigureHermes configures a Hermes Agent instance by writing .env and config.yaml
// inside the container, then restarting the gateway process.
func ConfigureHermes(cli *docker.Client, p HermesConfigureParams) error {
	const envPath = "/opt/data/.env"
	const configPath = "/opt/data/config.yaml"

	// Build .env additions.
	// Hermes uses provider-specific env vars (not OPENAI_API_KEY for everything).
	// OpenAI has no native provider in Hermes — we use model.api_key in config.yaml instead.
	// Provider may be empty for channel-only configuration.
	var envLines []string

	switch p.Provider {
	case "openai-codex":
		return fmt.Errorf("openai-codex (ChatGPT subscription) is not supported for Hermes instances. Please use a standard OpenAI API key model instead")
	case "": // Channel-only config — skip model credentials
	case "openai":
		// OpenAI has no native Hermes provider. API key is set via model.api_key below.
	case "anthropic":
		envLines = append(envLines, fmt.Sprintf("ANTHROPIC_API_KEY=%s", p.APIKey))
	case "google":
		envLines = append(envLines, fmt.Sprintf("GEMINI_API_KEY=%s", p.APIKey))
	case "deepseek":
		envLines = append(envLines, fmt.Sprintf("DEEPSEEK_API_KEY=%s", p.APIKey))
	}

	// Channel credentials
	if p.Channel == "discord" && p.ChannelToken != "" {
		envLines = append(envLines, fmt.Sprintf("DISCORD_BOT_TOKEN=%s", p.ChannelToken))
	} else if p.Channel == "telegram" && p.ChannelToken != "" {
		envLines = append(envLines, fmt.Sprintf("TELEGRAM_BOT_TOKEN=%s", p.ChannelToken))
	} else if p.Channel == "slack" && p.ChannelToken != "" {
		envLines = append(envLines, fmt.Sprintf("SLACK_BOT_TOKEN=%s", p.ChannelToken))
	}

	// Allow all users by default (ClawFleet manages access at the container level)
	envLines = append(envLines, "GATEWAY_ALLOW_ALL_USERS=true")

	// Write .env additions: remove any previous ClawFleet block, then append fresh.
	if len(envLines) > 0 {
		// Remove old ClawFleet-injected block (everything from the marker to EOF).
		_ = dockerExecAs(cli, p.ContainerID, "", []string{
			"sed", "-i", "/^# Configured by ClawFleet$/,$d", envPath,
		})
		envBlock := "\n# Configured by ClawFleet\n" + strings.Join(envLines, "\n") + "\n"
		if err := dockerExecAs(cli, p.ContainerID, "", []string{
			"bash", "-c", fmt.Sprintf("cat >> %s << 'CLAWFLEET_EOF'\n%sCLAWFLEET_EOF", envPath, envBlock),
		}); err != nil {
			return fmt.Errorf("writing .env: %w", err)
		}
	}

	// Set model in config.yaml via hermes CLI.
	// Hermes provider mapping:
	//   ClawFleet "openai"    → Hermes "openrouter" + model.base_url=api.openai.com + model.api_key
	//   ClawFleet "anthropic" → Hermes "anthropic"  + ANTHROPIC_API_KEY env
	//   ClawFleet "google"    → Hermes "gemini"     + GEMINI_API_KEY env
	//   ClawFleet "deepseek"  → Hermes "deepseek"   + DEEPSEEK_API_KEY env
	if p.Model != "" {
		model := p.Model

		var hermesProvider, baseURL, apiKey string
		switch p.Provider {
		case "openai":
			// Hermes has no native "openai" provider. Use "custom" provider with
			// direct API endpoint and key set in config.yaml. Verified working.
			hermesProvider = "custom"
			baseURL = "https://api.openai.com/v1"
			apiKey = p.APIKey
		case "anthropic":
			hermesProvider = "anthropic"
			if !strings.Contains(model, "/") {
				model = "anthropic/" + model
			}
		case "google":
			hermesProvider = "gemini"
			if !strings.Contains(model, "/") {
				model = "google/" + model
			}
		case "deepseek":
			hermesProvider = "deepseek"
			if !strings.Contains(model, "/") {
				model = "deepseek/" + model
			}
		}

		configCmds := []struct{ key, val string }{
			{"model.default", model},
			{"model.provider", hermesProvider},
			{"model.base_url", baseURL},
			{"model.api_key", apiKey},
		}
		for _, c := range configCmds {
			if err := dockerExecAs(cli, p.ContainerID, "", []string{
				"bash", "-c",
				fmt.Sprintf("source /opt/hermes/.venv/bin/activate && hermes config set %s '%s'", c.key, c.val),
			}); err != nil {
				return fmt.Errorf("setting %s: %w", c.key, err)
			}
		}
	}

	// Enable the channel platform in config.yaml
	if p.Channel != "" {
		platform := p.Channel
		if err := dockerExecAs(cli, p.ContainerID, "", []string{
			"bash", "-c",
			fmt.Sprintf("source /opt/hermes/.venv/bin/activate && hermes config set gateway.platforms.%s.enabled true", platform),
		}); err != nil {
			return fmt.Errorf("enabling %s platform: %w", platform, err)
		}
	}

	// Channel-only mode: if no model was configured by ClawFleet but the user
	// has Codex credentials from the Hermes Dashboard, auto-set the model provider
	// so the bot can actually respond. Without this, Codex login stores credentials
	// but model.provider stays "auto" and Hermes fails to find a provider.
	if p.Provider == "" {
		authCheck, _ := dockerExecOutputAs(cli, p.ContainerID, "", []string{
			"bash", "-c",
			`python3 -c "import json; d=json.load(open('/opt/data/auth.json')); print('codex' if d.get('credential_pool',{}).get('openai-codex') else 'none')"`,
		})
		if strings.TrimSpace(authCheck) == "codex" {
			// Codex credentials exist — set provider and a default model.
			for _, c := range []struct{ key, val string }{
				{"model.provider", "openai-codex"},
				{"model.default", "gpt-5.4-mini"},
			} {
				_ = dockerExecAs(cli, p.ContainerID, "", []string{
					"bash", "-c",
					fmt.Sprintf("source /opt/hermes/.venv/bin/activate && hermes config set %s '%s'", c.key, c.val),
				})
			}
		}
	}

	// Restart the container to pick up new .env and config changes.
	// The entrypoint will re-launch both dashboard and gateway.
	if err := cli.RestartContainer(p.ContainerID, 10); err != nil {
		return fmt.Errorf("restarting container: %w", err)
	}

	// Wait for gateway to come up (Hermes gateway doesn't have a /health endpoint
	// on a known port from outside, so we just wait a fixed duration)
	time.Sleep(15 * time.Second)

	return nil
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
