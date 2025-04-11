package linter

import (
	"context"
	"errors"
	"io"

	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/linter"
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
		Use:  "lint PATH...",
		Args: cobra.MinimumNArgs(1),
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
	ctx := context.Background()

	linterOpts := []linter.Option{
		linter.InputPaths(inputPaths),
		linter.WithCustomRules(opts.rules),
		linter.MaxConcurrency(opts.maxConcurrent),
	}

	if opts.debug {
		linterOpts = append(linterOpts, linter.Debug(cmd.ErrOrStderr()))
	}

	engine, err := linter.New(linterOpts...)
	if err != nil {
		return err
	}

	report, err := engine.Lint(ctx)
	if err != nil {
		return err
	}

	return opts.IO.Codec().Encode(cmd.OutOrStdout(), report)
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
