package linter

import (
	"context"
	"errors"
	"io"

	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/linter"
	"github.com/grafana/grafanactl/internal/resources/local"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type lintOpts struct {
	IO cmdio.Options

	debug         bool
	rules         []string
	maxConcurrent int
}

func (opts *lintOpts) validate(args []string) error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	if len(args) == 0 {
		return errors.New("at least one file or directory must be provided for linting")
	}

	if opts.maxConcurrent < 1 {
		return errors.New("max-concurrent must be greater than zero")
	}

	return nil
}

func (opts *lintOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("pretty", &reporterCodec{reporter: linter.PrettyReporter{}})
	opts.IO.RegisterCustomCodec("compact", &reporterCodec{reporter: linter.CompactReporter{}})
	opts.IO.DefaultFormat("pretty")

	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.debug, "debug", false, "Enable debug mode")
	flags.StringArrayVarP(&opts.rules, "rules", "r", nil, "Path to custom rules.")
	flags.IntVar(&opts.maxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
}

func lintCmd() *cobra.Command {
	opts := lintOpts{}

	cmd := &cobra.Command{
		Use:   "lint PATH...",
		Short: "Lint Grafana resources",
		Long:  "Lint Grafana resources.",
		Args:  cobra.MinimumNArgs(1),
		Example: `
	# Lint Grafana resources using builtin rules:

	grafanactl experimental linter lint ./resources

	# Lint specific files:

	grafanactl experimental linter lint ./resources/file.json ./resources/other.yaml

	# Display compact results:

	grafanactl experimental linter lint ./resources -o compact

	# Use custom rules:

	grafanactl experimental linter lint --rules ./custom-rules ./resources
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.validate(args); err != nil {
				return err
			}
			return lint(cmd, args, opts)
		},
	}

	opts.setup(cmd.Flags())
	return cmd
}

func lint(cmd *cobra.Command, inputPaths []string, opts lintOpts) error {
	linterOpts := []linter.Option{
		linter.InputPaths(inputPaths),
		linter.WithCustomRules(opts.rules),
		linter.MaxConcurrency(opts.maxConcurrent),
		linter.ResourceReader(&local.FSReader{
			Decoders:           format.Codecs(),
			MaxConcurrentReads: opts.maxConcurrent,
			StopOnError:        false,
		}),
	}

	if opts.debug {
		linterOpts = append(linterOpts, linter.Debug(cmd.ErrOrStderr()))
	}

	engine, err := linter.New(linterOpts...)
	if err != nil {
		return err
	}

	report, err := engine.Lint(cmd.Context())
	if err != nil {
		return err
	}

	return opts.IO.Encode(cmd.OutOrStdout(), report)
}

type reporterCodec struct {
	reporter linter.Reporter
}

func (c *reporterCodec) Encode(output io.Writer, input any) error {
	//nolint:forcetypeassert
	return c.reporter.Publish(context.Background(), output, input.(linter.Report))
}

func (c *reporterCodec) Decode(_ io.Reader, _ any) error {
	return errors.New("not supported")
}

func (c *reporterCodec) Format() format.Format {
	return "reporterCodec"
}
