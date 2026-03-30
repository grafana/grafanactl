package resources

import (
	"errors"
	"fmt"
	"io"
	"time"

	cmdconfig "github.com/grafana/grafanactl/cmd/grafanactl/config"
	cmdio "github.com/grafana/grafanactl/cmd/grafanactl/io"
	"github.com/grafana/grafanactl/internal/format"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/grafana/grafanactl/internal/resources/discovery"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/cli-runtime/pkg/printers"
	"k8s.io/client-go/metadata"
)

type getMetaOpts struct {
	IO            cmdio.Options
	LabelSelector string
}

func (opts *getMetaOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("text", &partialMetaTableCodec{wide: false})
	opts.IO.RegisterCustomCodec("wide", &partialMetaTableCodec{wide: true})
	opts.IO.DefaultFormat("text")

	flags.StringVarP(&opts.LabelSelector, "selector", "l", "", "Filter resources by label selector (e.g. -l key=value,other=value)")

	opts.IO.BindFlags(flags)
}

func (opts *getMetaOpts) Validate() error {
	return opts.IO.Validate()
}

func getMetaCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &getMetaOpts{}

	cmd := &cobra.Command{
		Use:   "get-meta RESOURCE_SELECTOR",
		Args:  cobra.ExactArgs(1),
		Short: "Get partial object metadata for Grafana resources",
		Long:  "Get partial object metadata (name, namespace, labels, annotations) for Grafana resources.",
		Example: `
	# All instances of a resource type:

	grafanactl resources get-meta dashboards

	# One or more specific instances:

	grafanactl resources get-meta dashboards/foo
	grafanactl resources get-meta dashboards/foo,bar

	# Long kind format with version:

	grafanactl resources get-meta dashboards.v1alpha1.dashboard.grafana.app
	grafanactl resources get-meta dashboards.v1alpha1.dashboard.grafana.app/foo
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			codec, err := opts.IO.Codec()
			if err != nil {
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
				Selectors:            sels,
				PreferredVersionOnly: true,
			})
			if err != nil {
				return err
			}

			if len(filters) != 1 {
				return fmt.Errorf("expected exactly one resource type, got %d", len(filters))
			}

			filter := filters[0]

			mdClient, err := metadata.NewForConfig(&cfg.Config)
			if err != nil {
				return err
			}

			gvr := filter.Descriptor.GroupVersionResource()
			rc := mdClient.Resource(gvr).Namespace(cfg.Namespace)

			var list metav1.PartialObjectMetadataList

			switch filter.Type {
			case resources.FilterTypeAll:
				result, err := rc.List(ctx, metav1.ListOptions{})
				if err != nil {
					return err
				}

				list = *result

			case resources.FilterTypeSingle, resources.FilterTypeMultiple:
				g, ctx := errgroup.WithContext(ctx)
				items := make([]metav1.PartialObjectMetadata, len(filter.ResourceUIDs))

				for i, name := range filter.ResourceUIDs {
					g.Go(func() error {
						item, err := rc.Get(ctx, name, metav1.GetOptions{})
						if err != nil {
							return err
						}

						items[i] = *item
						return nil
					})
				}

				if err := g.Wait(); err != nil {
					return err
				}

				list.Items = items
			}

			return codec.Encode(cmd.OutOrStdout(), &list)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

type partialMetaTableCodec struct {
	wide bool
}

func (c *partialMetaTableCodec) Format() format.Format {
	if c.wide {
		return "wide"
	}

	return "text"
}

func (c *partialMetaTableCodec) Encode(output io.Writer, input any) error {
	list, ok := input.(*metav1.PartialObjectMetadataList)
	if !ok {
		return fmt.Errorf("expected *metav1.PartialObjectMetadataList, got %T", input)
	}

	table := &metav1.Table{
		ColumnDefinitions: []metav1.TableColumnDefinition{
			{
				Name:        "NAME",
				Type:        "string",
				Format:      "name",
				Priority:    0,
				Description: "The name of the resource.",
			},
			{
				Name:        "NAMESPACE",
				Type:        "string",
				Priority:    0,
				Description: "The namespace of the resource.",
			},
			{
				Name:        "AGE",
				Type:        "string",
				Format:      "date-time",
				Priority:    0,
				Description: "The age of the resource.",
			},
		},
	}

	if c.wide {
		table.ColumnDefinitions = append(table.ColumnDefinitions, metav1.TableColumnDefinition{
			Name:        "LABELS",
			Type:        "string",
			Priority:    0,
			Description: "The labels of the resource.",
		})
	}

	for i := range list.Items {
		item := &list.Items[i]
		age := duration.HumanDuration(time.Since(item.CreationTimestamp.Time))

		var row metav1.TableRow
		if c.wide {
			row = metav1.TableRow{
				Cells:  []any{item.Name, item.Namespace, age, labels.FormatLabels(item.Labels)},
				Object: runtime.RawExtension{Object: item},
			}
		} else {
			row = metav1.TableRow{
				Cells:  []any{item.Name, item.Namespace, age},
				Object: runtime.RawExtension{Object: item},
			}
		}

		table.Rows = append(table.Rows, row)
	}

	printer := printers.NewTablePrinter(printers.PrintOptions{
		Wide: c.wide,
	})

	return printer.PrintObj(table, output)
}

func (c *partialMetaTableCodec) Decode(io.Reader, any) error {
	return errors.New("partialMetaTableCodec does not support decoding")
}
