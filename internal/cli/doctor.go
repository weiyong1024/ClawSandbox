package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Run a local preflight check and show the next step",
	Example: "  clawsandbox doctor",
	RunE:    runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println("ClawSandbox Doctor")
	fmt.Println()

	cli, err := container.NewClient()
	if err != nil {
		fmt.Println("Docker: NOT READY")
		fmt.Println("  ClawSandbox could not reach your local Docker engine.")
		fmt.Println()
		fmt.Println("Current path: blocked before startup")
		fmt.Println("Estimated wait: 1-2 minutes after Docker starts")
		fmt.Println("Next step:")
		fmt.Println("  1. Start Docker Desktop (or your Docker engine).")
		fmt.Println("  2. Wait until `docker version` succeeds.")
		fmt.Println("  3. Run `clawsandbox doctor` again.")
		return err
	}

	fmt.Println("Docker: READY")
	fmt.Println("  Your local Docker engine is reachable.")
	fmt.Println()

	imageRef := cfg.ImageRef()
	imageExists, err := container.ImageExists(cli, imageRef)
	if err != nil {
		return err
	}

	if imageExists {
		fmt.Println("Image: READY")
		fmt.Printf("  Local image `%s` is already available.\n", imageRef)
		fmt.Println()
		fmt.Println("Current path: ready to create claws now")
		fmt.Println("Estimated wait: Dashboard starts in seconds")
		fmt.Println("Next step:")
		fmt.Println("  1. Run `clawsandbox dashboard serve`.")
		fmt.Println("  2. Open `http://localhost:8080`.")
		fmt.Println("  3. Create your first claw.")
		return nil
	}

	fmt.Println("Image: NOT PRESENT LOCALLY")
	fmt.Printf("  Local image `%s` is not available on this machine yet.\n", imageRef)
	fmt.Println()
	fmt.Println("Current path: build the local image first")
	fmt.Println("Estimated wait: the first local build usually takes several minutes")
	fmt.Println("Next step:")
	fmt.Println("  1. Run `clawsandbox build`.")
	fmt.Println("  2. Run `clawsandbox dashboard serve`.")
	fmt.Println("  3. Open `http://localhost:8080` and create your first claw.")
	return nil
}
