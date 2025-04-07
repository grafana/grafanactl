package resources

import (
	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type pushOpts struct {
	ContinueOnError bool
	Kinds           []string
}

func (opts *pushOpts) setup(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pushing resources even if an error occurs")
	flags.StringArrayVar(&opts.Kinds, "kind", opts.Kinds, "Resource kinds to push. If omitted, all supported kinds will be pulled")
}

func pushCmd(configOpts *cmdconfig.Options) *cobra.Command {
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
			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			resourcesPath := args[0]

			cmd.Printf("Pushing resources from '%s' to context '%s'\n", resourcesPath, cfg.CurrentContext)

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
