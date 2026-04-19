package cmd

import (
	"fmt"
	"time"

	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/spf13/cobra"
)

var (
	silenceDuration string
	silenceReason   string

	silenceStateNew = state.New

	// SilenceCmd creates a silence for an alert fingerprint.
	SilenceCmd = &cobra.Command{
		Use:   "silence <fingerprint>",
		Short: "Silence an alert by fingerprint",
		Long:  "Creates a silence rule that suppresses notifications for the given alert fingerprint.",
		Args:  cobra.ExactArgs(1),
		RunE:  runSilence,
	}
)

func init() {
	SilenceCmd.Flags().StringVar(&silenceDuration, "for", "1h", "Silence duration (e.g. 1h, 30m, 24h)")
	SilenceCmd.Flags().StringVar(&silenceReason, "reason", "", "Reason for silencing")
}

func runSilence(cmd *cobra.Command, args []string) error {
	fingerprint := args[0]

	dur, err := time.ParseDuration(silenceDuration)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", silenceDuration, err)
	}

	stateMgr := silenceStateNew()
	now := time.Now()

	silences, err := stateMgr.LoadSilences()
	if err != nil {
		return fmt.Errorf("failed to load silences: %w", err)
	}

	silences = append(silences, state.Silence{
		Fingerprint: fingerprint,
		CreatedAt:   now,
		ExpiresAt:   now.Add(dur),
		Reason:      silenceReason,
	})

	if err := stateMgr.SaveSilences(silences); err != nil {
		return fmt.Errorf("failed to save silence: %w", err)
	}

	result := map[string]any{
		"fingerprint": fingerprint,
		"silenced":    true,
		"expires_at":  now.Add(dur).Format(time.RFC3339),
		"reason":      silenceReason,
	}

	return outputResult(result)
}
