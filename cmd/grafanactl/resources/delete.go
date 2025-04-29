package resources

import (
	"context"
	"errors"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	"github.com/grafana/grafanactl/cmd/grafanactl/fail"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/grafana/grafanactl/internal/resources/local"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type deleteOpts struct {
	StopOnError   bool
	Force         bool
	MaxConcurrent int
	DryRun        bool
	Directories   []string
}

func (opts *deleteOpts) setup(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pulling resources when an error occurs")
	flags.IntVar(&opts.MaxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
	flags.BoolVar(&opts.Force, "force", opts.Force, "Delete all resources of the specified resource types")
	flags.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "If set, the delete operation will be simulated")
	flags.StringSliceVarP(&opts.Directories, "directory", "d", nil, "Directories on disk containing the resources to delete")
}

func (opts *deleteOpts) Validate(args []string) error {
	if opts.MaxConcurrent < 1 {
		return errors.New("max-concurrent must be greater than zero")
	}

	if len(args) == 0 && len(opts.Directories) == 0 {
		return errors.New("either --directory or resource selectors need to be specified")
	}

	return nil
}

func deleteCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &deleteOpts{}

	cmd := &cobra.Command{
		Use:   "delete [RESOURCE_SELECTOR]...",
		Args:  cobra.ArbitraryArgs,
		Short: "Delete resources from Grafana",
		Long:  "Delete resources from Grafana.",
		Example: `
	# Delete a single dashboard
	grafanactl resources delete dashboards/some-dashboard

	# Delete multiple dashboards
	grafanactl resources delete dashboards/some-dashboard,other-dashboard

	# Delete a dashboard and a folder
	grafanactl resources delete dashboards/some-dashboard folders/some-folder

	# Delete every dashboard
	grafanactl resources delete dashboards --force

	# Delete every resource defined in the given directory
	grafanactl resources delete -d ./unwanted-resources/

	# Delete every dashboard defined in the given directory
	grafanactl resources delete -d ./unwanted-resources/ dashboard
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := opts.Validate(args); err != nil {
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

			if !opts.Force && !sels.HasNamedSelectorsOnly() {
				return fail.DetailedError{
					Summary: "Invalid resource selector",
					Details: "Expected a resource selector targeting named resources only. Example: dashboard/some-dashboard",
					Suggestions: []string{
						"Specify the --force flag to force the deletion.",
					},
				}
			}

			var res resources.Resources

			// Load resources by selectors only
			if len(opts.Directories) == 0 {
				fetchRes, err := fetchResources(ctx, fetchRequest{
					Config:      cfg,
					StopOnError: opts.StopOnError,
				}, args)
				if err != nil {
					return err
				}

				res = fetchRes.Resources
			} else {
				// Load resources from the filesystem
				res = *resources.NewResources()
				if err := loadResourcesFromDirectories(ctx, cfg, &res, opts, sels); err != nil {
					return err
				}
			}

			if opts.DryRun {
				cmdio.Info(cmd.OutOrStdout(), "Dry-run mode enabled")
			}

			// Delete!
			deleter, err := remote.NewDeleter(ctx, cfg)
			if err != nil {
				return err
			}

			req := remote.DeleteRequest{
				Resources:      &res,
				MaxConcurrency: opts.MaxConcurrent,
				StopOnError:    opts.StopOnError,
				DryRun:         opts.DryRun,
			}

			summary, err := deleter.Delete(ctx, req)
			if err != nil {
				return err
			}

			// Reporting time.
			printer := cmdio.Success
			if summary.FailedCount != 0 {
				printer = cmdio.Warning
				if summary.DeletedCount == 0 {
					printer = cmdio.Error
				}
			}

			printer(cmd.OutOrStdout(), "%d resources deleted, %d errors", summary.DeletedCount, summary.FailedCount)

			return nil
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func loadResourcesFromDirectories(ctx context.Context, cfg config.NamespacedRESTConfig, res *resources.Resources, opts *deleteOpts, selectors resources.Selectors) error {
	reg, err := discovery.NewDefaultRegistry(ctx, cfg)
	if err != nil {
		return err
	}

	reader := local.FSReader{
		Decoders:           format.Codecs(),
		MaxConcurrentReads: opts.MaxConcurrent,
		StopOnError:        opts.StopOnError,
	}

	filters, err := reg.MakeFilters(selectors)
	if err != nil {
		return err
	}

	return reader.Read(ctx, res, filters, opts.Directories)
}
