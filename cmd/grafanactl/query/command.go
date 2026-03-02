package query

import (
	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/spf13/cobra"
)

// Command returns the query command group.
func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Execute queries against Grafana datasources",
		Long:  "Execute queries against Grafana datasources such as Prometheus and Loki.",
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(prometheusCmd(configOpts))

	return cmd
}
