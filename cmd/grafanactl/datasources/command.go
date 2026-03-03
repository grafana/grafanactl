package datasources

import (
	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/spf13/cobra"
)

// Command returns the datasources command group.
func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "datasources",
		Short: "Manage Grafana datasources",
		Long:  "List and get information about Grafana datasources.",
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(listCmd(configOpts))
	cmd.AddCommand(getCmd(configOpts))
	cmd.AddCommand(prometheusCmd(configOpts))
	cmd.AddCommand(lokiCmd(configOpts))

	return cmd
}
