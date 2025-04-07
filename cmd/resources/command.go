package resources

import (
	"os"
	"path"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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

type serveOpts struct {
	Address string
	Port    int
}

func (opts *serveOpts) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Address, "address", "0.0.0.0", "Address to bind")
	flags.IntVar(&opts.Port, "port", 8080, "Port on which the server will listen")
}

func serveCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &serveOpts{}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve Grafana resources locally",
		Long: `Serve Grafana resources locally.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " serve",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			cmd.Printf("Serving resources with context: %s\n", cfg.CurrentContext)

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}
