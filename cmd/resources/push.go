package resources

import (
	"errors"
	"fmt"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
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
	flags.StringSliceVarP(&opts.Directories, "directory", "d", []string{defaultResourcesDir}, "Directories on disk from which to read the resources to push.")
	flags.IntVar(&opts.MaxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pushing resources when an error occurs")
	flags.BoolVar(&opts.OverwriteExisting, "overwrite", opts.OverwriteExisting, "Overwrite existing resources")
	flags.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "If set, the push operation will be simulated, without actually creating or updating any resources.")
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
		Example: fmt.Sprintf(`
  Everything:

  %[1]s resources push

  All instances for a given kind(s):

  %[1]s resources push dashboards
  %[1]s resources push dashboards folders

  Single resource kind, one or more resource instances:

  %[1]s resources push dashboards/foo
  %[1]s resources push dashboards/foo,bar

  Single resource kind, long kind format:

  %[1]s resources push dashboard.dashboards/foo
  %[1]s resources push dashboard.dashboards/foo,bar

  Single resource kind, long kind format with version:

  %[1]s resources push dashboards.v1alpha1.dashboard.grafana.app/foo
  %[1]s resources push dashboards.v1alpha1.dashboard.grafana.app/foo,bar

  Multiple resource kinds, one or more resource instances:

  %[1]s resources push dashboards/foo folders/qux
  %[1]s resources push dashboards/foo,bar folders/qux,quux

  Multiple resource kinds, long kind format:

  %[1]s resources push dashboard.dashboards/foo folder.folders/qux
  %[1]s resources push dashboard.dashboards/foo,bar folder.folders/qux,quux

  Multiple resource kinds, long kind format with version:

  %[1]s resources push dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux
`, binaryName),
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
				return parseSelectorErr(err)
			}

			reader := resources.FSReader{
				Decoders:           format.Codecs(),
				MaxConcurrentReads: opts.MaxConcurrent,
				StopOnError:        opts.StopOnError,
			}

			resourcesList := resources.NewResources()

			if err := reader.Read(ctx, resourcesList, opts.Directories); err != nil {
				return err
			}

			pusher, err := resources.NewPusher(ctx, cfg)
			if err != nil {
				return clientInitErr(err)
			}

			req := resources.PushRequest{
				Selectors:         sels,
				Resources:         resourcesList,
				MaxConcurrency:    opts.MaxConcurrent,
				StopOnError:       opts.StopOnError,
				OverwriteExisting: opts.OverwriteExisting,
				DryRun:            opts.DryRun,
			}

			return pusher.Push(ctx, req)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
