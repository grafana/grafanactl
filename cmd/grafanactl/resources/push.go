package resources

import (
	"errors"

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

type pushOpts struct {
	Directories       []string
	MaxConcurrent     int
	StopOnError       bool
	OverwriteExisting bool
	DryRun            bool
}

func (opts *pushOpts) setup(flags *pflag.FlagSet) {
	flags.StringSliceVarP(&opts.Directories, "directory", "d", []string{defaultResourcesDir}, "Directories on disk from which to read the resources to push")
	flags.IntVar(&opts.MaxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pushing resources when an error occurs")
	flags.BoolVar(&opts.OverwriteExisting, "overwrite", opts.OverwriteExisting, "Overwrite existing resources")
	flags.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "If set, the push operation will be simulated, without actually creating or updating any resources")
}

func (opts *pushOpts) Validate() error {
	if len(opts.Directories) == 0 {
		return errors.New("at least one directory is required")
	}

	if opts.MaxConcurrent < 1 {
		return errors.New("max-concurrent must be greater than zero")
	}

	return nil
}

func pushCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &pushOpts{}

	cmd := &cobra.Command{
		Use:   "push [RESOURCE_SELECTOR]...",
		Args:  cobra.ArbitraryArgs,
		Short: "Push resources to Grafana",
		Long:  "Push resources to Grafana using a specific format. See examples below for more details.",
		Example: `
	# Everything:

	grafanactl resources push

	# All instances for a given kind(s):

	grafanactl resources push dashboards
	grafanactl resources push dashboards folders

	# Single resource kind, one or more resource instances:

	grafanactl resources push dashboards/foo
	grafanactl resources push dashboards/foo,bar

	# Single resource kind, long kind format:

	grafanactl resources push dashboard.dashboards/foo
	grafanactl resources push dashboard.dashboards/foo,bar

	# Single resource kind, long kind format with version:

	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo
	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo,bar

	# Multiple resource kinds, one or more resource instances:

	grafanactl resources push dashboards/foo folders/qux
	grafanactl resources push dashboards/foo,bar folders/qux,quux

	# Multiple resource kinds, long kind format:

	grafanactl resources push dashboard.dashboards/foo folder.folders/qux
	grafanactl resources push dashboard.dashboards/foo,bar folder.folders/qux,quux

	# Multiple resource kinds, long kind format with version:

	grafanactl resources push dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

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

			filters, err := reg.MakeFilters(sels)
			if err != nil {
				return err
			}

			reader := local.FSReader{
				Decoders:           format.Codecs(),
				MaxConcurrentReads: opts.MaxConcurrent,
				StopOnError:        opts.StopOnError,
			}

			resourcesList := resources.NewResources()

			if err := reader.Read(ctx, resourcesList, filters, opts.Directories); err != nil {
				return err
			}

			pusher, err := remote.NewDefaultPusher(ctx, cfg)
			if err != nil {
				return err
			}

			req := remote.PushRequest{
				Resources:         resourcesList,
				MaxConcurrency:    opts.MaxConcurrent,
				StopOnError:       opts.StopOnError,
				OverwriteExisting: opts.OverwriteExisting,
				DryRun:            opts.DryRun,
			}

			summary, err := pusher.Push(ctx, req)
			if err != nil {
				return err
			}

			printer := cmdio.Success
			if summary.FailedCount != 0 {
				printer = cmdio.Warning
				if summary.PushedCount == 0 {
					printer = cmdio.Error
				}
			}

			printer(cmd.OutOrStdout(), "%d resources pushed, %d errors", summary.PushedCount, summary.FailedCount)

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
