package resources

import (
	"errors"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/resources/local"
	"github.com/grafana/grafanactl/internal/resources/process"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultResourcesPath = "./resources"
)

type pullOpts struct {
	IO             cmdio.Options
	StopOnError    bool
	IncludeManaged bool
	Path           string
}

func (opts *pullOpts) setup(flags *pflag.FlagSet) {
	// Bind all the flags
	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pulling resources when an error occurs")
	flags.StringVarP(&opts.Path, "path", "p", defaultResourcesPath, "Path on disk in which the resources will be written")
	flags.BoolVar(
		&opts.IncludeManaged,
		"include-managed",
		opts.IncludeManaged,
		"Include resources managed by tools other than grafanactl",
	)
}

func (opts *pullOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	if opts.Path == "" {
		return errors.New("--path is required")
	}

	return nil
}

func pullCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &pullOpts{}

	cmd := &cobra.Command{
		Use:   "pull [RESOURCE_SELECTOR]...",
		Args:  cobra.ArbitraryArgs,
		Short: "Pull resources from Grafana",
		Long:  "Pull resources from Grafana using a specific format. See examples below for more details.",
		Example: `
	# Everything:

	grafanactl resources pull

	# All instances for a given kind(s):

	grafanactl resources pull dashboards
	grafanactl resources pull dashboards folders

	# Single resource kind, one or more resource instances:

	grafanactl resources pull dashboards/foo
	grafanactl resources pull dashboards/foo,bar

	# Single resource kind, long kind format:

	grafanactl resources pull dashboard.dashboards/foo
	grafanactl resources pull dashboard.dashboards/foo,bar

	# Single resource kind, long kind format with version:

	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo
	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo,bar

	# Multiple resource kinds, one or more resource instances:

	grafanactl resources pull dashboards/foo folders/qux
	grafanactl resources pull dashboards/foo,bar folders/qux,quux

	# Multiple resource kinds, long kind format:

	grafanactl resources pull dashboard.dashboards/foo folder.folders/qux
	grafanactl resources pull dashboard.dashboards/foo,bar folder.folders/qux,quux

	# Multiple resource kinds, long kind format with version:

	grafanactl resources pull dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := opts.Validate(); err != nil {
				return err
			}

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			res, err := fetchResources(cmd.Context(), fetchRequest{
				Config: cfg,
				// Strip server fields from the resources.
				// This includes fields like `resourceVersion`, `uid`, etc.
				Processors: []remote.Processor{
					&process.ServerFieldsStripper{},
				},
				ExcludeManaged: !opts.IncludeManaged,
				StopOnError:    opts.StopOnError,
			}, args)
			if err != nil {
				return err
			}

			writer := local.FSWriter{
				Path:        opts.Path,
				Namer:       local.GroupResourcesByKind(opts.IO.OutputFormat),
				Encoder:     codec,
				StopOnError: opts.StopOnError,
			}

			if err := writer.Write(ctx, &res.Resources); err != nil {
				return err
			}

			cmdio.Success(cmd.OutOrStdout(), "%d resources pulled", res.Resources.Len())

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
