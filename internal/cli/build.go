package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the OpenClaw sandbox Docker image",
	Long: `Build the clawsandbox/openclaw Docker image locally.
This is required once before running 'clawsandbox create'.
The build downloads ~1.4 GB and may take several minutes.`,
	Example: "  clawsandbox build",
	RunE:    runBuild,
}

func runBuild(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	imageRef := cfg.ImageRef()
	fmt.Fprintf(os.Stdout, "Building image %s ...\n\n", imageRef)

	if err := container.Build(cli, imageRef, os.Stdout); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("\nImage %s built successfully.\n", imageRef)
	fmt.Println("Run 'clawsandbox create <N>' to deploy your fleet.")
	return nil
}
