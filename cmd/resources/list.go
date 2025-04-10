package resources

import (
	"fmt"
	"text/tabwriter"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/spf13/cobra"
)

func listCmd(configOpts *cmdconfig.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Args:  cobra.NoArgs,
		Short: "List available Grafana API resources",
		Long:  "List available Grafana API resources.",
		Example: fmt.Sprintf(`
  %[1]s resources list
`, binaryName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			cfg, err := configOpts.LoadRESTConfig(ctx)
			if err != nil {
				return err
			}

			reg, err := resources.NewDefaultDiscoveryRegistry(ctx, cfg)
			if err != nil {
				return clientInitErr(err)
			}

			// TODO: refactor this to return a k8s object list,
			// e.g. APIResourceList, or unstructured.UnstructuredList.
			// That way we can use the same code for rendering as for `resources get`.
			res, err := reg.Resources(ctx, false)
			if err != nil {
				return clientInitErr(err)
			}

			out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
			fmt.Fprintf(out, "GROUP\tVERSION\tKIND\n")
			for _, r := range res {
				fmt.Fprintf(out, "%s\t%s\t%s\n", r.Group, r.Version, r.Kind)
			}

			return out.Flush()
		},
	}

	return cmd
}
