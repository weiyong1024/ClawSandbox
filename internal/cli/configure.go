package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var configureCmd = &cobra.Command{
	Use:   "configure <name>",
	Short: "Configure an OpenClaw instance with LLM provider and chat channel",
	Args:  cobra.ExactArgs(1),
	Example: `  clawsandbox configure claw-1 \
    --provider anthropic --api-key sk-ant-... --model claude-sonnet-4-6 \
    --channel telegram --channel-token 123456:ABC...`,
	RunE: runConfigure,
}

func init() {
	f := configureCmd.Flags()
	f.String("provider", "anthropic", "LLM provider (anthropic, openai, etc.)")
	f.String("api-key", "", "LLM API key")
	f.String("model", "", "Model ID (e.g. claude-sonnet-4-6)")
	f.String("channel", "", "Chat channel (telegram, discord, slack, etc.)")
	f.String("channel-token", "", "Channel bot token")

	_ = configureCmd.MarkFlagRequired("api-key")
}

func runConfigure(cmd *cobra.Command, args []string) error {
	name := args[0]

	store, err := state.Load()
	if err != nil {
		return err
	}

	inst := store.Get(name)
	if inst == nil {
		return fmt.Errorf("instance %s not found", name)
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	// Ensure instance is running
	status, _, _ := container.Status(cli, inst.ContainerID)
	if status != "running" {
		fmt.Printf("Instance %s is not running, starting it... ", name)
		if err := container.Start(cli, inst.ContainerID); err != nil {
			fmt.Println("failed")
			return fmt.Errorf("starting instance: %w", err)
		}
		store.SetStatus(name, "running")
		_ = store.Save()
		fmt.Println("done")
	}

	provider, _ := cmd.Flags().GetString("provider")
	apiKey, _ := cmd.Flags().GetString("api-key")
	model, _ := cmd.Flags().GetString("model")
	channel, _ := cmd.Flags().GetString("channel")
	channelToken, _ := cmd.Flags().GetString("channel-token")

	fmt.Printf("Configuring %s (this may take up to 30s while the gateway starts)...\n", name)

	if err := container.Configure(cli, container.ConfigureParams{
		ContainerID:  inst.ContainerID,
		Provider:     provider,
		APIKey:       apiKey,
		Model:        model,
		Channel:      channel,
		ChannelToken: channelToken,
	}); err != nil {
		return fmt.Errorf("configure failed: %w", err)
	}

	fmt.Printf("Instance %s configured successfully\n", name)
	if model != "" {
		fmt.Printf("  Model: %s\n", model)
	}
	if channel != "" {
		fmt.Printf("  Channel: %s\n", channel)
	}

	return nil
}
