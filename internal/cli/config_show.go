package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/weiyong1024/clawsandbox/internal/config"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Short:   "Show current configuration",
	Example: "  clawsandbox config",
	RunE:    runConfigShow,
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dataDir, _ := config.DataDir()
	configPath := ""
	if dataDir != "" {
		configPath = dataDir + "/config.yaml"
	}

	fmt.Println("# ClawSandbox Configuration")
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("# Source: %s\n", configPath)
		} else {
			fmt.Printf("# Source: defaults (no config file at %s)\n", configPath)
		}
	}
	fmt.Printf("# Data dir: %s\n", dataDir)
	fmt.Println()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}
