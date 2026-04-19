package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/AmadlaOrg/lighthouse/plugin"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	pluginsNew = plugin.New

	pluginsOutputFlag string
	pluginsHeryFlag   bool

	// PluginsCmd lists all discovered lighthouse plugins.
	PluginsCmd = &cobra.Command{
		Use:   "plugins",
		Short: "List discovered lighthouse plugins",
		RunE:  runPlugins,
	}
)

func init() {
	PluginsCmd.Flags().StringVarP(&pluginsOutputFlag, "output", "o", "table", "Output format: table, json, yaml")
	PluginsCmd.Flags().BoolVar(&pluginsHeryFlag, "hery", false, "Wrap output in HERY envelope (_type, _body)")
}

type heryEnvelope struct {
	Type string `json:"_type" yaml:"_type"`
	Body any    `json:"_body" yaml:"_body"`
}

type pluginRow struct {
	Plugin      string `json:"plugin" yaml:"plugin"`
	Channel     string `json:"channel" yaml:"channel"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description" yaml:"description"`
}

func runPlugins(cmd *cobra.Command, args []string) error {
	svc := pluginsNew()

	plugins, err := svc.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	if len(plugins) == 0 {
		fmt.Fprintln(os.Stderr, "No lighthouse plugins found in PATH.")
		return nil
	}

	var rows []pluginRow
	for _, name := range plugins {
		info, err := svc.GetInfo(name)
		if err != nil {
			rows = append(rows, pluginRow{
				Plugin:      name,
				Channel:     "?",
				Version:     "?",
				Description: fmt.Sprintf("error: %v", err),
			})
			continue
		}
		rows = append(rows, pluginRow{
			Plugin:      name,
			Channel:     info.Channel,
			Version:     info.Version,
			Description: info.Description,
		})
	}

	var data any = rows
	if pluginsHeryFlag {
		data = heryEnvelope{
			Type: "amadla.org/entity/tools/plugins@v1.0.0",
			Body: rows,
		}
	}

	switch pluginsOutputFlag {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "yaml":
		return yaml.NewEncoder(os.Stdout).Encode(data)
	default:
		table := tablewriter.NewWriter(os.Stdout)
		table.Header("Plugin", "Channel", "Version", "Description")
		for _, r := range rows {
			table.Append(r.Plugin, r.Channel, r.Version, r.Description)
		}
		table.Render()
		return nil
	}
}
