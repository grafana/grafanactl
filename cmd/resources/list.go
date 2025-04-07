package resources

import (
	"fmt"
	"text/tabwriter"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type listOpts struct {
	// none so far
}

func (opts *listOpts) BindFlags(*pflag.FlagSet) {
	// none so far
}

func listCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &listOpts{}

	cmd := &cobra.Command{
		Use:   "list",
		Args:  cobra.ArbitraryArgs,
		Short: "List available Grafana API resources",
		Long:  "List available Grafana API resources.",
		Example: fmt.Sprintf(`
  %[1]s resources list
`, binaryName),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfig(cmd.Context())
			if err != nil {
				return err
			}

			rest, err := config.NewNamespacedRESTConfig(*cfg.GetCurrentContext())
			if err != nil {
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not parse config",
					Details: "The config provided is invalid",
					Suggestions: []string{
						"Make sure that you are passing in a valid config",
					},
				}
			}

			reg, err := resources.NewDefaultDiscoveryRegistry(rest)
			if err != nil {
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not discover resources from the API",
					Details: "The API may not be reachable or the server may not support the discovery API",
					Suggestions: []string{
						"Make sure that the API server is running and accessible",
						"Try using a different context or check your configuration",
					},
				}
			}

			res, err := reg.Resources(cmd.Context(), false)
			if err != nil {
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not discover resources from the API",
					Details: "The API may not be reachable or the server may not support the discovery API",
					Suggestions: []string{
						"Make sure that the API server is running and accessible",
						"Try using a different context or check your configuration",
					},
				}
			}

			// TODO: add formatters (yaml, json, etc.) / outputters (stdout, file, filetree, etc.)
			out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
			fmt.Fprintf(out, "GROUP\tVERSION\tKIND\n")
			for _, r := range res {
				fmt.Fprintf(out, "%s\t%s\t%s\n", r.Group, r.Version, r.Kind)
			}

			return out.Flush()
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}
