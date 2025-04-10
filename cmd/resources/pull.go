package resources

import (
	"errors"
	"fmt"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	defaultResourcesDir = "./resources"
)

type pullOpts struct {
	IO          cmdio.Options
	StopOnError bool
	Directory   string
}

func (opts *pullOpts) setup(flags *pflag.FlagSet) {
	// Bind all the flags
	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pulling resources when an error occurs")
	flags.StringVarP(&opts.Directory, "directory", "d", defaultResourcesDir, "Directory on disk in which the resources will be written.")
}

func (opts *pullOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	if opts.Directory == "" {
		return errors.New("--directory is required")
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
		Example: fmt.Sprintf(`
  Everything:

  %[1]s resources pull

  All instances for a given kind(s):

  %[1]s resources pull dashboards
  %[1]s resources pull dashboards folders

  Single resource kind, one or more resource instances:

  %[1]s resources pull dashboards/foo
  %[1]s resources pull dashboards/foo,bar

  Single resource kind, long kind format:

  %[1]s resources pull dashboard.dashboards/foo
  %[1]s resources pull dashboard.dashboards/foo,bar

  Single resource kind, long kind format with version:

  %[1]s resources pull dashboards.v1alpha1.dashboard.grafana.app/foo
  %[1]s resources pull dashboards.v1alpha1.dashboard.grafana.app/foo,bar

  Multiple resource kinds, one or more resource instances:

  %[1]s resources pull dashboards/foo folders/qux
  %[1]s resources pull dashboards/foo,bar folders/qux,quux

  Multiple resource kinds, long kind format:

  %[1]s resources pull dashboard.dashboards/foo folder.folders/qux
  %[1]s resources pull dashboard.dashboards/foo,bar folder.folders/qux,quux

  Multiple resource kinds, long kind format with version:

  %[1]s resources pull dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux
`, binaryName),
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
				Config:      cfg,
				StopOnError: opts.StopOnError,
			}, args)
			if err != nil {
				return err
			}

			writer := resources.FSWriter{
				Directory:   opts.Directory,
				Namer:       resources.GroupResourcesByKind(opts.IO.OutputFormat),
				Encoder:     codec,
				StopOnError: opts.StopOnError,
			}

			return writer.Write(ctx, res.Resources)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}
