package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/version"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the OpenClaw sandbox Docker image",
	Long: `Build the OpenClaw sandbox Docker image locally.
This is the recommended first-run path today and is required whenever
the local image is missing.
It also supports offline use or customization.`,
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

	// Also tag as :latest when building a versioned image
	if version.ImageTag() != "latest" {
		latestRef := fmt.Sprintf("%s:latest", cfg.Image.Name)
		if err := container.TagImage(cli, imageRef, cfg.Image.Name, "latest"); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to tag as %s: %v\n", latestRef, err)
		} else {
			fmt.Printf("Also tagged as %s\n", latestRef)
		}
	}

	fmt.Printf("\nImage %s built successfully.\n", imageRef)
	fmt.Println("Run 'clawsandbox create <N>' to deploy your fleet.")
	return nil
}
