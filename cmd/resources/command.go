package resources

import (
	"os"
	"path"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var binaryName = path.Base(os.Args[0])

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

	return cmd
}
