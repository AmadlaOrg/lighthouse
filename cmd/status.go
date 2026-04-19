package cmd

import (
	"fmt"
	"os"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	statusStateNew = state.New

	// StatusCmd shows the current alert status.
	StatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Show active alerts and silences",
		RunE:  runStatus,
	}
)

func runStatus(cmd *cobra.Command, args []string) error {
	stateMgr := statusStateNew()

	alerts, err := stateMgr.LoadAlerts()
	if err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	silences, err := stateMgr.LoadSilences()
	if err != nil {
		return fmt.Errorf("failed to load silences: %w", err)
	}

	if len(alerts) == 0 && len(silences) == 0 {
		fmt.Fprintln(os.Stderr, "No active alerts or silences.")
		return nil
	}

	if len(alerts) > 0 {
		fmt.Println("Active Alerts:")
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Fingerprint", "Source", "Name", "Severity", "Status", "Count", "Last Seen")

		for _, a := range alerts {
			fp := a.Fingerprint
			if len(fp) > 12 {
				fp = fp[:12] + "..."
			}
			table.Append(fp, a.Source, a.Name, a.Severity, a.Status,
				fmt.Sprintf("%d", a.Count), a.LastSeen.Format("15:04:05"))
		}
		table.Render()
	}

	if len(silences) > 0 {
		fmt.Println("\nActive Silences:")
		silenceTable := tablewriter.NewWriter(os.Stdout)
		silenceTable.Header("Fingerprint", "Expires At", "Reason")

		for _, s := range silences {
			fp := s.Fingerprint
			if len(fp) > 12 {
				fp = fp[:12] + "..."
			}
			silenceTable.Append(fp, s.ExpiresAt.Format("2006-01-02 15:04:05"), s.Reason)
		}
		silenceTable.Render()
	}

	return nil
}
