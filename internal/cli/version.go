package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Injected at build time via ldflags. See Makefile / .goreleaser.yml.
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
)

var versionShort bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of ClawSandbox",
	Run: func(cmd *cobra.Command, args []string) {
		if versionShort {
			fmt.Println(Version)
			return
		}
		fmt.Printf("clawsandbox %s\n", Version)
		fmt.Printf("  commit:    %s\n", GitCommit)
		fmt.Printf("  built:     %s\n", BuildDate)
		fmt.Printf("  go:        %s\n", runtime.Version())
		fmt.Printf("  platform:  %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Print version number only")
}
