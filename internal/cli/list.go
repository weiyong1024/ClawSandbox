package cli

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/container"
	"github.com/weiyong1024/clawsandbox/internal/state"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "status"},
	Short:   "List all claw instances and their status",
	Example: "  clawsandbox list\n  clawsandbox ls",
	RunE:    runList,
}

func runList(cmd *cobra.Command, args []string) error {
	store, err := state.Load()
	if err != nil {
		return err
	}

	instances := store.Snapshot()
	if len(instances) == 0 {
		fmt.Println("No instances found. Run 'clawsandbox create <N>' to get started.")
		return nil
	}

	cli, err := container.NewClient()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tDESKTOP\tGATEWAY\tUPTIME")
	fmt.Fprintln(w, "────\t──────\t───────\t───────\t──────")

	for _, inst := range instances {
		status, startedAt, _ := container.Status(cli, inst.ContainerID)
		store.SetStatus(inst.Name, status)

		desktop := fmt.Sprintf("http://localhost:%d", inst.Ports.NoVNC)
		gateway := fmt.Sprintf(":%d", inst.Ports.Gateway)
		uptime := "—"
		if status == "running" && !startedAt.IsZero() {
			uptime = formatUptime(startedAt)
		}

		if status != "running" {
			desktop = "—"
			gateway = "—"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", inst.Name, status, desktop, gateway, uptime)
	}

	w.Flush()
	_ = store.Save()
	return nil
}

func formatUptime(since time.Time) string {
	d := time.Since(since)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	return fmt.Sprintf("%dd %dh", days, h)
}
