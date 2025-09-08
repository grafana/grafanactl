package resources

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/local"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type validateOpts struct {
	IO cmdio.Options

	Paths         []string
	MaxConcurrent int
	StopOnError   bool
}

func (opts *validateOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("text", &validationTableCodec{})
	opts.IO.DefaultFormat("text")

	opts.IO.BindFlags(flags)

	flags.StringSliceVarP(&opts.Paths, "path", "p", []string{defaultResourcesPath}, "Paths on disk from which to read the resources.")
	flags.IntVar(&opts.MaxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop validating resources when an error occurs")
}

func (opts *validateOpts) Validate() error {
	if len(opts.Paths) == 0 {
		return errors.New("at least one path is required")
	}

	if opts.MaxConcurrent < 1 {
		return errors.New("max-concurrent must be greater than zero")
	}

	return nil
}

func validateCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &validateOpts{}

	cmd := &cobra.Command{
		Use:   "validate [RESOURCE_SELECTOR]...",
		Args:  cobra.ArbitraryArgs,
		Short: "Validate resources",
		Long: `Validate resources.

This command validates its inputs against a remote Grafana instance.
`,
		Example: `
	# Validate all resources in the default directory
	grafanactl resources validate

	# Validate a single resource kind
	grafanactl resources validate dashboards

	# Validate a multiple resource kinds
	grafanactl resources validate dashboards folders

	# Displaying validation results as YAML
	grafanactl resources validate -o yaml

	# Displaying validation results as JSON
	grafanactl resources validate -o json
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			sels, err := resources.ParseSelectors(args)
			if err != nil {
				return err
			}

			reg, err := discovery.NewDefaultRegistry(ctx, cfg)
			if err != nil {
				return err
			}

			filters, err := reg.MakeFilters(discovery.MakeFiltersOptions{
				Selectors: sels,
			})
			if err != nil {
				return err
			}

			reader := local.FSReader{
				Decoders:           format.Codecs(),
				MaxConcurrentReads: opts.MaxConcurrent,
				StopOnError:        opts.StopOnError,
			}

			resourcesList := resources.NewResources()

			if err := reader.Read(ctx, resourcesList, filters, opts.Paths); err != nil {
				return err
			}

			pusher, err := remote.NewDefaultPusher(ctx, cfg)
			if err != nil {
				return err
			}

			req := remote.PushRequest{
				Resources:        resourcesList,
				MaxConcurrency:   opts.MaxConcurrent,
				StopOnError:      opts.StopOnError,
				DryRun:           true,
				NoPushFailureLog: true,
			}

			summary, err := pusher.Push(ctx, req)
			if err != nil {
				return err
			}

			if summary.FailedCount == 0 && opts.IO.OutputFormat == "text" {
				cmdio.Success(cmd.OutOrStdout(), "No errors found.")
				return nil
			}

			if opts.IO.OutputFormat == "text" {
				return codec.Encode(cmd.OutOrStdout(), summary)
			}

			printableSummary := struct {
				Failures []map[string]string `json:"failures" yaml:"failures"`
			}{
				Failures: make([]map[string]string, 0),
			}

			for _, failure := range summary.Failures {
				printableSummary.Failures = append(printableSummary.Failures, map[string]string{
					"file":  failure.Resource.SourcePath(),
					"error": failure.Error.Error(),
				})
			}

			return codec.Encode(cmd.OutOrStdout(), printableSummary)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

type validationTableCodec struct{}

func (c *validationTableCodec) Format() format.Format {
	return "text"
}

func (c *validationTableCodec) Encode(output io.Writer, input any) error {
	//nolint:forcetypeassert
	summary := input.(*remote.PushSummary)

	tab := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)

	fmt.Fprintf(tab, "FILE\tERROR\n")
	for _, failure := range summary.Failures {
		file := failure.Resource.SourcePath()
		fmt.Fprintf(tab, "%s\t%s\n", file, failure.Error)
	}

	return tab.Flush()
}

func (c *validationTableCodec) Decode(io.Reader, any) error {
	return errors.New("codec does not support decoding")
}
