package resources

import (
	"os"
	"path"

	"github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//nolint:gochecknoglobals
var binaryName = path.Base(os.Args[0])

func Command() *cobra.Command {
	configOpts := &config.Options{}

	cmd := &cobra.Command{
		Use:   "resources",
		Short: "Manipulate Grafana resources",
		Long: `Manipulate Grafana resources.

TODO: more information.
`,
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(pullCmd(configOpts))
	cmd.AddCommand(pushCmd(configOpts))
	cmd.AddCommand(serveCmd(configOpts))

	return cmd
}

type pullOpts struct {
	ContinueOnError bool
	Kinds           []string
}

func (opts *pullOpts) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pulling resources even if an error occurs")
	flags.StringArrayVar(&opts.Kinds, "kind", opts.Kinds, "Resource kinds to pull. If omitted, all supported kinds will be pulled")
}

func pullCmd(configOpts *config.Options) *cobra.Command {
	opts := &pullOpts{}

	cmd := &cobra.Command{
		Use:   "pull RESOURCES_PATH",
		Args:  cobra.ExactArgs(1),
		Short: "Pull resources from Grafana",
		Long: `Pull resources from Grafana.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " resources pull",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			resourcesPath := args[0]

			cmd.Printf("Pulling resources into '%s' from context '%s'\n", resourcesPath, cfg.CurrentContext)

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}

type pushOpts struct {
	ContinueOnError bool
	Kinds           []string
}

func (opts *pushOpts) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pushing resources even if an error occurs")
	flags.StringArrayVar(&opts.Kinds, "kind", opts.Kinds, "Resource kinds to push. If omitted, all supported kinds will be pulled")
}

func pushCmd(configOpts *config.Options) *cobra.Command {
	opts := &pushOpts{}

	cmd := &cobra.Command{
		Use:     "push RESOURCES_PATH",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"apply"},
		Short:   "Push resources to Grafana",
		Long: `Push resources from Grafana.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " resources push",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			resourcesPath := args[0]

			cmd.Printf("Pushing resources from '%s' to context '%s'\n", resourcesPath, cfg.CurrentContext)

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

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

func serveCmd(configOpts *config.Options) *cobra.Command {
	opts := &serveOpts{}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve Grafana resources locally",
		Long: `Serve Grafana resources locally.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " serve",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfig()
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
