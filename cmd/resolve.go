package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AmadlaOrg/lighthouse/alert"
	"github.com/AmadlaOrg/lighthouse/config"
	"github.com/AmadlaOrg/lighthouse/engine"
	"github.com/AmadlaOrg/lighthouse/plugin"
	"github.com/AmadlaOrg/lighthouse/state"
	"github.com/spf13/cobra"
)

var (
	resolveFilePath string

	// ResolveCmd processes an alert resolution.
	ResolveCmd = &cobra.Command{
		Use:   "resolve",
		Short: "Send a resolution notification",
		Long:  "Reads an alert from file or stdin and processes it as resolved.",
		RunE:  runResolve,
	}
)

func init() {
	ResolveCmd.Flags().StringVarP(&resolveFilePath, "file", "f", "", "Alert input file (JSON or YAML; use '-' for stdin)")
	_ = ResolveCmd.MarkFlagRequired("file")
}

func runResolve(cmd *cobra.Command, args []string) error {
	var input io.Reader
	if resolveFilePath == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(resolveFilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: cannot open file: %v\n", err)
			os.Exit(2)
		}
		defer f.Close()
		input = f
	}

	data, err := io.ReadAll(input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: cannot read input: %v\n", err)
		os.Exit(2)
	}

	a, err := alert.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse alert: %w", err)
	}

	// Force resolved status
	a.Status = "resolved"

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	stateMgr := state.New()
	pluginSvc := plugin.New()
	eng := engine.New(cfg, stateMgr, pluginSvc)

	result, err := eng.Process(a, time.Now())
	if err != nil {
		return fmt.Errorf("failed to process resolution: %w", err)
	}

	return outputResult(result)
}
