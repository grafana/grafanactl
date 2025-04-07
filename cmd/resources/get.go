package resources

import (
	"fmt"
	"io"
	"strings"
	"time"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	cmdio "github.com/grafana/grafanactl/cmd/io"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
)

type getOpts struct {
	IO              cmdio.Options
	ContinueOnError bool
}

func (opts *getOpts) setup(flags *pflag.FlagSet) {
	// Setup some additional formatting options
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pulling resources even if an error occurs")
	opts.IO.RegisterCustomFormat("text", tableFormatter(false))
	opts.IO.RegisterCustomFormat("wide", tableFormatter(true))
	opts.IO.DefaultFormat("text")

	// Bind all the flags
	opts.IO.BindFlags(flags)
}

func (opts *getOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	return nil
}

func getCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &getOpts{}

	cmd := &cobra.Command{
		Use:   "get RESOURCES_PATHS",
		Args:  cobra.ArbitraryArgs,
		Short: "Get resources from Grafana",
		Long:  "Get resources from Grafana using a specific format. See examples below for more details.",
		Example: fmt.Sprintf(`
  Everything:

  %[1]s resources get dashboards/foo

  All instances for a given kind(s):

  %[1]s resources get dashboards
  %[1]s resources get dashboards folders

  Single resource kind, one or more resource instances:

  %[1]s resources get dashboards/foo
  %[1]s resources get dashboards/foo,bar

  Single resource kind, long kind format:

  %[1]s resources get dashboard.dashboards/foo
  %[1]s resources get dashboard.dashboards/foo,bar

  Single resource kind, long kind format with version:

  %[1]s resources get dashboards.v1alpha1.dashboard.grafana.app/foo
  %[1]s resources get dashboards.v1alpha1.dashboard.grafana.app/foo,bar

  Multiple resource kinds, one or more resource instances:

  %[1]s resources get dashboards/foo folders/qux
  %[1]s resources get dashboards/foo,bar folders/qux,quux

  Multiple resource kinds, long kind format:

  %[1]s resources get dashboard.dashboards/foo folder.folders/qux
  %[1]s resources get dashboard.dashboards/foo,bar folder.folders/qux,quux

  Multiple resource kinds, long kind format with version:

  %[1]s resources get dashboards.v1alpha1.dashboard.grafana.app/foo folders.v1alpha1.folder.grafana.app/qux
`, binaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			res, err := fetchResources(cmd.Context(), fetchOpts{
				Config:          cfg,
				ContinueOnError: opts.ContinueOnError,
			}, args)
			if err != nil {
				return err
			}

			// Avoid printing a list of results if a single resource is being pulled,
			// and we are not using the table output format.
			if res.IsSingleTarget &&
				len(res.Resources.Items) == 1 &&
				opts.IO.OutputFormat != "text" &&
				opts.IO.OutputFormat != "wide" {
				return opts.IO.Format(cmd.OutOrStdout(), res.Resources.Items[0])
			}

			return opts.IO.Format(cmd.OutOrStdout(), res.Resources)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

func tableFormatter(wide bool) func(output io.Writer, input any) error {
	return func(output io.Writer, input any) error {
		//nolint:forcetypeassert
		items := input.(unstructured.UnstructuredList)

		// TODO: support per-kind column definitions.
		//
		// Read more about type & format here:
		// https://github.com/OAI/OpenAPI-Specification/blob/main/versions/2.0.md#data-types
		//
		// Priority is 0-based (from most important to least important)
		// and controls whether columns are omitted in (wide: false) tables.
		table := &metav1.Table{
			ColumnDefinitions: []metav1.TableColumnDefinition{
				{
					Name:        "KIND",
					Type:        "string",
					Priority:    0,
					Description: "The kind of the resource.",
				},
				{
					Name:        "NAME",
					Type:        "string",
					Format:      "name",
					Priority:    0,
					Description: "The name of the resource.",
				},
				{
					Name:        "NAMESPACE",
					Priority:    0,
					Description: "The namespace of the resource.",
				},
				{
					Name:        "AGE",
					Type:        "string",
					Format:      "date-time",
					Priority:    1,
					Description: "The age of the resource.",
				},
			},
		}

		for _, r := range items.Items {
			age := duration.HumanDuration(time.Since(r.GetCreationTimestamp().Time))

			table.Rows = append(table.Rows, metav1.TableRow{
				Cells: []interface{}{
					formatKind(r.GroupVersionKind(), wide),
					r.GetName(),
					r.GetNamespace(),
					age,
				},
				Object: runtime.RawExtension{
					Object: &r,
				},
			})
		}

		printer := printers.NewTablePrinter(printers.PrintOptions{
			Wide:       wide,
			ShowLabels: wide,
			// TODO: sorting doesn't actually do anything,
			// though it is supported in the options.
			// SortBy:     "name",
		})

		return printer.PrintObj(table, output)
	}
}

// TODO: we need to change the format of data the puller returns,
// to include the API metadata for each resource.
func formatKind(gvk schema.GroupVersionKind, wide bool) string {
	plural := strings.ToLower(gvk.Kind) + "s"
	if wide {
		return fmt.Sprintf("%s.%s.%s", plural, gvk.Version, gvk.Group)
	}

	return plural
}
