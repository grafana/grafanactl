package resources

import (
	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "resources",
		Short: "Manipulate Grafana resources",
		Long: `Manipulate Grafana resources.

TODO: more information.
`,
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(getCmd(configOpts))
	cmd.AddCommand(listCmd(configOpts))
	cmd.AddCommand(pullCmd(configOpts))
	cmd.AddCommand(pushCmd(configOpts))
	cmd.AddCommand(serveCmd(configOpts))
	cmd.AddCommand(validateCmd(configOpts))

	return cmd
}
