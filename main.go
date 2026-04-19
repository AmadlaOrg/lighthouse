package main

import (
	"fmt"
	"os"

	"github.com/AmadlaOrg/lighthouse/cmd"
	"github.com/spf13/cobra"
)

const (
	appName = "lighthouse"
	version = "1.0.0"
)

var rootCmd = &cobra.Command{
	Use:     appName,
	Short:   "Intelligent notification/alerting CLI with lighthouse-* plugins",
	Version: version,
}

func init() {
	rootCmd.AddCommand(cmd.NotifyCmd)
	rootCmd.AddCommand(cmd.ResolveCmd)
	rootCmd.AddCommand(cmd.SilenceCmd)
	rootCmd.AddCommand(cmd.StatusCmd)
	rootCmd.AddCommand(cmd.PluginsCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
