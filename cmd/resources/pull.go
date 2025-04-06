package resources

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/grafana/grafana-app-sdk/logging"
	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"
)

type pullOpts struct {
	IO cmdio.Options

	ContinueOnError bool
	Directory       string
}

func (opts *pullOpts) setup(flags *pflag.FlagSet) {
	// Setup some additional formatting options
	opts.IO.RegisterCustomFormat("text", formatResourcesAsText)
	opts.IO.RegisterCustomFormat("wide", formatResourcesAsWideText)

	opts.IO.DefaultFormat("text")

	// Bind all the flags
	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pulling resources even if an error occurs")
	flags.StringVarP(&opts.Directory, "directory", "d", "", "Directory on disk in which the resources will be written. If left empty, nothing will be written on disk and resource details will be printed on stdout")
}

func (opts *pullOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	if opts.Directory != "" && (opts.IO.OutputFormat == "text" || opts.IO.OutputFormat == "wide") {
		return errors.New("--directory and --output=text|wide are mutually exclusive")
	}

	return nil
}

func pullCmd(logger logging.Logger, configOpts *cmdconfig.Options) *cobra.Command {
	opts := &pullOpts{}

	cmd := &cobra.Command{
		Use:   "pull RESOURCES_PATHS",
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
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig(logger)
			if err != nil {
				return err
			}

			pull, err := resources.NewPuller(logger, *cfg.GetCurrentContext())
			if err != nil {
				// TODO: is this error actually related to what `resources.NewPuller()` does?
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not parse pull command(s)",
					Details: "One or more of the provided resource paths are invalid",
					Suggestions: []string{
						"Make sure that your are passing in valid resource paths",
					},
				}
			}

			var (
				res              *unstructured.UnstructuredList
				singlePullTarget bool
				perr             error
			)
			if len(args) == 0 {
				res, perr = pull.PullAll(cmd.Context())
			} else {
				invalidCommandErr := &resources.InvalidCommandError{}
				cmds, err := resources.ParsePullCommands(args)
				if err != nil && errors.As(err, invalidCommandErr) {
					return fail.DetailedError{
						Parent:  err,
						Summary: "Could not parse pull command(s)",
						Details: fmt.Sprintf("Failed to parse command '%s'", invalidCommandErr.Command),
						Suggestions: []string{
							"Make sure that your are passing in valid resource paths",
						},
					}
				} else if err != nil {
					return err
				}

				singlePullTarget = cmds.HasSingleTarget()
				res, perr = pull.Pull(cmd.Context(), resources.PullerRequest{
					Commands:        cmds,
					ContinueOnError: opts.ContinueOnError,
				})
			}

			if perr != nil {
				return fail.DetailedError{
					Parent:  perr,
					Summary: "Could not pull resource(s) from the API",
					Details: "One or more resources could not be pulled from the API",
					Suggestions: []string{
						"Make sure that your are passing in valid resource paths",
					},
				}
			}

			if opts.Directory != "" {
				writer := resources.FSWriter{
					Logger:          logger,
					Directory:       opts.Directory,
					Namer:           resources.GroupResourcesByKind(opts.IO.OutputFormat),
					Formatter:       opts.IO.Format,
					ContinueOnError: opts.ContinueOnError,
				}

				return writer.Write(res)
			}

			// Avoid printing a list of results if a single resource is being pulled
			if len(res.Items) != 0 && singlePullTarget && opts.IO.OutputFormat != "text" && opts.IO.OutputFormat != "wide" {
				return opts.IO.Format(cmd.OutOrStdout(), res.Items[0])
			}

			return opts.IO.Format(cmd.OutOrStdout(), res)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func formatResourcesAsText(output io.Writer, input any) error {
	//nolint:forcetypeassert
	items := input.(*unstructured.UnstructuredList)

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
	fmt.Fprintf(out, "KIND\tNAME\tAGE\n")
	for _, r := range items.Items {
		gvk := r.GroupVersionKind()
		age := duration.HumanDuration(time.Since(r.GetCreationTimestamp().Time))

		fmt.Fprintf(out, "%s\t%s\t%s\n", gvk.Kind, r.GetName(), age)
	}

	return out.Flush()
}

func formatResourcesAsWideText(output io.Writer, input any) error {
	//nolint:forcetypeassert
	items := input.(*unstructured.UnstructuredList)

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
	fmt.Fprintf(out, "GROUP\tVERSION\tKIND\tNAMESPACE\tNAME\tAGE\n")
	for _, r := range items.Items {
		gvk := r.GroupVersionKind()
		age := duration.HumanDuration(time.Since(r.GetCreationTimestamp().Time))

		fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%st%s\n", gvk.Group, gvk.Version, gvk.Kind, r.GetNamespace(), r.GetName(), age)
	}

	return out.Flush()
}
