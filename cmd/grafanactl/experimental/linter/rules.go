package linter

import (
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/linter"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type rulesOpts struct {
	IO cmdio.Options

	debug bool
	rules []string
}

func (opts *rulesOpts) validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	return nil
}

func (opts *rulesOpts) setup(flags *pflag.FlagSet) {
	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.debug, "debug", false, "Enable debug mode")
	flags.StringArrayVarP(&opts.rules, "rules", "r", nil, "Path to custom rules.")
}

func rulesCmd() *cobra.Command {
	opts := rulesOpts{}

	cmd := &cobra.Command{
		Use:   "rules",
		Args:  cobra.NoArgs,
		Short: "List available linter rules",
		Long:  "List available linter rules.",
		Example: `
	# List built-in rules:

	grafanactl experimental linter rules

	# List built-in and custom rules:

	grafanactl experimental linter rules -v ./custom-rules
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validate(); err != nil {
				return err
			}
			return listRules(cmd, args, opts)
		},
	}

	opts.setup(cmd.Flags())
	return cmd
}

func listRules(cmd *cobra.Command, inputPaths []string, opts rulesOpts) error {
	linterOpts := []linter.Option{
		linter.InputPaths(inputPaths),
		linter.WithCustomRules(opts.rules),
	}

	if opts.debug {
		linterOpts = append(linterOpts, linter.Debug(cmd.ErrOrStderr()))
	}

	engine, err := linter.New(linterOpts...)
	if err != nil {
		return err
	}

	rules, err := engine.Rules(cmd.Context())
	if err != nil {
		return err
	}

	return opts.IO.Encode(cmd.OutOrStdout(), rules)
}
