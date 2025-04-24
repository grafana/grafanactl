package resources

import (
	"errors"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type deleteOpts struct {
	StopOnError   bool
	All           bool
	MaxConcurrent int
	DryRun        bool
}

func (opts *deleteOpts) setup(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.StopOnError, "stop-on-error", opts.StopOnError, "Stop pulling resources when an error occurs")
	flags.IntVar(&opts.MaxConcurrent, "max-concurrent", 10, "Maximum number of concurrent operations")
	flags.BoolVar(&opts.All, "all", opts.All, "Delete all resources of the specified resource types")
	flags.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "If set, the delete operation will be simulated")
}

func (opts *deleteOpts) Validate() error {
	if opts.MaxConcurrent < 1 {
		return errors.New("max-concurrent must be greater than zero")
	}

	return nil
}

func deleteCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &deleteOpts{}

	cmd := &cobra.Command{
		Use:   "delete RESOURCE_SELECTOR...",
		Args:  cobra.MinimumNArgs(1),
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
	grafanactl resources delete dashboards --all
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			res, err := fetchResources(ctx, fetchRequest{
				Config:               cfg,
				StopOnError:          opts.StopOnError,
				ExpectNamedSelectors: !opts.All,
			}, args)
			if err != nil {
				return err
			}

			toDelete, err := resources.NewResourcesFromUnstructured(res.Resources)
			if err != nil {
				return err
			}

			if opts.DryRun {
				cmdio.Info(cmd.OutOrStdout(), "Dry-run mode enabled")
			}

			deleter, err := remote.NewDeleter(ctx, cfg)
			if err != nil {
				return err
			}

			req := remote.DeleteRequest{
				Resources:      toDelete,
				MaxConcurrency: opts.MaxConcurrent,
				StopOnError:    opts.StopOnError,
				DryRun:         opts.DryRun,
			}

			summary, err := deleter.Delete(ctx, req)
			if err != nil {
				return err
			}

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
