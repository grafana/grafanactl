package alerts

import (
	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/spf13/cobra"
)

// Command returns the alerts command group.
func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Manage Grafana alert rules",
		Long:  "Manage Grafana alert rules via the Provisioning API.",
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(exportCmd(configOpts))
	cmd.AddCommand(getCmd(configOpts))
	cmd.AddCommand(historyCmd(configOpts))
	cmd.AddCommand(instancesCmd(configOpts))
	cmd.AddCommand(listCmd(configOpts))
	cmd.AddCommand(noiseReportCmd(configOpts))
	cmd.AddCommand(searchCmd(configOpts))

	return cmd
}
