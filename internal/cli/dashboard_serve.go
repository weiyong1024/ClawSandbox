package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/config"
	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/web"
)

var dashboardServePort int
var dashboardServeHost string

var dashboardServeCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Start the ClawSandbox Web Dashboard server",
	Example: "  clawsandbox dashboard serve\n  clawsandbox dashboard serve --port 9090 --host 0.0.0.0",
	RunE:    runDashboardServe,
}

func init() {
	dashboardServeCmd.Flags().IntVar(&dashboardServePort, "port", 8080, "HTTP listen port")
	dashboardServeCmd.Flags().StringVar(&dashboardServeHost, "host", "127.0.0.1", "HTTP listen host")
}

func runDashboardServe(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", dashboardServeHost, dashboardServePort)
	srv := web.NewServer(cli, cfg, addr)
	return srv.ListenAndServe()
}
