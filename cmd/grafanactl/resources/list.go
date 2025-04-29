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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type listOpts struct {
	IO cmdio.Options
}

func (opts *listOpts) setup(flags *pflag.FlagSet) {
	opts.IO.RegisterCustomCodec("text", &tabCodec{wide: false})
	opts.IO.RegisterCustomCodec("wide", &tabCodec{wide: true})
	opts.IO.DefaultFormat("text")

	opts.IO.BindFlags(flags)
}

func (opts *listOpts) Validate() error {
	return opts.IO.Validate()
}

func listCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &listOpts{}

	cmd := &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List available Grafana API resources",
		Long:  "List available Grafana API resources.",
		Example: `
	grafanactl resources list
`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			codec, err := opts.IO.Codec()
			if err != nil {
				return err
			}

			if opts.IO.OutputFormat != "text" && opts.IO.OutputFormat != "wide" {
				return fmt.Errorf("unsupported output format: %s", opts.IO.OutputFormat)
			}

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			reg, err := discovery.NewDefaultRegistry(ctx, cfg)
			if err != nil {
				return err
			}

			// TODO: refactor this to return a k8s object list,
			// e.g. APIResourceList, or unstructured.UnstructuredList.
			// That way we can use the same code for rendering as for `resources get`.
			res := reg.SupportedResources().Sorted()
			return codec.Encode(cmd.OutOrStdout(), res)
		},
	}

	opts.setup(cmd.Flags())

	return cmd
}

type tabCodec struct {
	wide bool
}

func (c *tabCodec) Format() format.Format {
	if c.wide {
		return "wide"
	}

	return "text"
}

func (c *tabCodec) Encode(output io.Writer, input any) error {
	descs, ok := input.(resources.Descriptors)
	if !ok {
		return fmt.Errorf("expected resources.Descriptors, got %T", input)
	}

	out := tabwriter.NewWriter(output, 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
	if c.wide {
		fmt.Fprintf(out, "GROUP\tVERSION\tPLURAL\tSINGULAR\tKIND\n")
	} else {
		fmt.Fprintf(out, "GROUP\tVERSION\tPLURAL\n")
	}

	for _, r := range descs {
		gv := r.GroupVersion
		if c.wide {
			fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", gv.Group, gv.Version, r.Plural, r.Singular, r.Kind)
		} else {
			fmt.Fprintf(out, "%s\t%s\t%s\n", gv.Group, gv.Version, r.Plural)
		}
	}

	return out.Flush()
}

func (c *tabCodec) Decode(io.Reader, any) error {
	return errors.New("tab codec does not support decoding")
}
