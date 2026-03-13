package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/weiyong1024/clawsandbox/internal/state"
)

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all snapshots",
	Args:  cobra.NoArgs,
	RunE:  runSnapshotList,
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	snapStore, err := state.LoadSnapshots()
	if err != nil {
		return err
	}

	snapshots := snapStore.List()
	if len(snapshots) == 0 {
		fmt.Println("No snapshots found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSOURCE\tCREATED\tSIZE")
	for _, s := range snapshots {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			s.Name,
			s.SourceInstance,
			s.CreatedAt.Format("2006-01-02 15:04"),
			formatSize(s.SizeBytes),
		)
	}
	return w.Flush()
}

func formatSize(bytes int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case bytes >= gb:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(gb))
	case bytes >= mb:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(mb))
	case bytes >= kb:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(kb))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
