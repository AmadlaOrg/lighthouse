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
	notifyFilePath string

	// NotifyCmd processes an incoming alert.
	NotifyCmd = &cobra.Command{
		Use:   "notify",
		Short: "Process an incoming alert notification",
		Long:  "Reads an alert from file or stdin and processes it through the intelligent suppression pipeline.",
		RunE:  runNotify,
	}
)

func init() {
	NotifyCmd.Flags().StringVarP(&notifyFilePath, "file", "f", "", "Alert input file (JSON or YAML; use '-' for stdin)")
	_ = NotifyCmd.MarkFlagRequired("file")
}

func runNotify(cmd *cobra.Command, args []string) error {
	var input io.Reader
	if notifyFilePath == "-" {
		input = os.Stdin
	} else {
		f, err := os.Open(notifyFilePath)
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	stateMgr := state.New()
	pluginSvc := plugin.New()
	eng := engine.New(cfg, stateMgr, pluginSvc)

	now := time.Now()

	result, err := eng.Process(a, now)
	if err != nil {
		return fmt.Errorf("failed to process alert: %w", err)
	}

	// Also flush any due groups
	groupResults, _ := eng.FlushGroups(now)
	for _, gr := range groupResults {
		outputResult(gr)
	}

	return outputResult(result)
}
