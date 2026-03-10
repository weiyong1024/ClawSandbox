package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var destroyPurge bool

var destroyCmd = &cobra.Command{
	Use:     "destroy <name|all>",
	Short:   "Destroy a claw instance (data is kept by default)",
	Args:    cobra.ExactArgs(1),
	Example: "  clawsandbox destroy claw-1\n  clawsandbox destroy all --purge",
	RunE:    runDestroy,
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyPurge, "purge", false, "Also delete instance data from disk")
}

func runDestroy(cmd *cobra.Command, args []string) error {
	store, err := state.Load()
	if err != nil {
		return err
	}

	targets := resolveTargets(store, args[0])
	if len(targets) == 0 {
		return fmt.Errorf("no instance found: %s", args[0])
	}

	if len(targets) > 1 || destroyPurge {
		purgeNote := ""
		if destroyPurge {
			purgeNote = " (and their data)"
		}
		fmt.Printf("About to destroy %d instance(s)%s. Continue? [y/N] ", len(targets), purgeNote)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	dockerCli, err := container.NewClient()
	if err != nil {
		return err
	}

	dataDir, err := config.DataDir()
	if err != nil && destroyPurge {
		return fmt.Errorf("cannot determine data dir for purge: %w", err)
	}

	for _, inst := range targets {
		fmt.Printf("Destroying %s ... ", inst.Name)

		if err := container.Remove(dockerCli, inst.ContainerID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}

		store.Remove(inst.Name)
		if err := store.Save(); err != nil {
			return fmt.Errorf("saving state: %w", err)
		}

		// Release any channel assigned to this instance so it becomes available again.
		if assets, err := state.LoadAssets(); err == nil {
			assets.ReleaseChannelByInstance(inst.Name)
			_ = assets.SaveAssets()
		}

		if destroyPurge {
			instanceDir := filepath.Join(dataDir, "data", inst.Name)
			if err := os.RemoveAll(instanceDir); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not remove data dir: %v\n", err)
			}
		}

		fmt.Println("✓")
	}

	return nil
}
