package resources

import (
	"fmt"
	"os"
	"path"
	"text/tabwriter"

	cmdconfig "github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/grafana/grafanactl/internal/resources"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//nolint:gochecknoglobals
var binaryName = path.Base(os.Args[0])

func Command() *cobra.Command {
	configOpts := &cmdconfig.Options{}

	cmd := &cobra.Command{
		Use:   "resources",
		Short: "Manipulate Grafana resources",
		Long: `Manipulate Grafana resources.

TODO: more information.
`,
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(listCmd(configOpts))
	cmd.AddCommand(pullCmd(configOpts))
	cmd.AddCommand(pushCmd(configOpts))
	cmd.AddCommand(serveCmd(configOpts))

	return cmd
}

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
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			cfgCtx, err := cfg.GetCurrentContext()
			if err != nil {
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not get config for the context",
					Details: "The config provided does not have an active context set",
					Suggestions: []string{
						"Make sure that you are passing in a valid config",
						"Make sure that you are using a context that exists in the config",
					},
				}
			}

			rest, err := config.NewNamespacedRESTConfig(*cfgCtx)
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

type pullOpts struct {
	ContinueOnError bool
	Kinds           []string
}

func (opts *pullOpts) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pulling resources even if an error occurs")
}

func pullCmd(configOpts *cmdconfig.Options) *cobra.Command {
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
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			cfgCtx, err := cfg.GetCurrentContext()
			if err != nil {
				return fail.DetailedError{
					Parent:  err,
					Summary: "Could not get config for the context",
					Details: "The config provided does not have an active context set",
					Suggestions: []string{
						"Make sure that you are passing in a valid config",
						"Make sure that you are using a context that exists in the config",
					},
				}
			}

			pull, err := resources.NewPuller(*cfgCtx)
			if err != nil {
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
				res  []unstructured.Unstructured
				perr error
			)
			if len(args) == 0 {
				res, perr = pull.PullAll(cmd.Context())
			} else {
				res, perr = pull.Pull(cmd.Context(), resources.PullerCommand{
					Commands: args,
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

			// TODO: add formatters (yaml, json, etc.) / outputters (stdout, file, filetree, etc.)
			out := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', tabwriter.TabIndent|tabwriter.DiscardEmptyColumns)
			fmt.Fprintf(out, "GROUP\tVERSION\tKIND\tNAMESPACE\tNAME\n")
			for _, r := range res {
				gvk := r.GroupVersionKind()
				fmt.Fprintf(out, "%s\t%s\t%s\t%s\t%s\n", gvk.Group, gvk.Version, gvk.Kind, r.GetNamespace(), r.GetName())
			}

			return out.Flush()
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}

type pushOpts struct {
	ContinueOnError bool
	Kinds           []string
}

func (opts *pushOpts) BindFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&opts.ContinueOnError, "continue-on-error", opts.ContinueOnError, "Continue pushing resources even if an error occurs")
	flags.StringArrayVar(&opts.Kinds, "kind", opts.Kinds, "Resource kinds to push. If omitted, all supported kinds will be pulled")
}

func pushCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &pushOpts{}

	cmd := &cobra.Command{
		Use:     "push RESOURCES_PATH",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"apply"},
		Short:   "Push resources to Grafana",
		Long: `Push resources from Grafana.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " resources push",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			resourcesPath := args[0]

			cmd.Printf("Pushing resources from '%s' to context '%s'\n", resourcesPath, cfg.CurrentContext)

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}

type serveOpts struct {
	Address string
	Port    int
}

func (opts *serveOpts) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.Address, "address", "0.0.0.0", "Address to bind")
	flags.IntVar(&opts.Port, "port", 8080, "Port on which the server will listen")
}

func serveCmd(configOpts *cmdconfig.Options) *cobra.Command {
	opts := &serveOpts{}

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve Grafana resources locally",
		Long: `Serve Grafana resources locally.

TODO: more information.
`,
		Example: "\n\t" + binaryName + " serve",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			cmd.Printf("Serving resources with context: %s\n", cfg.CurrentContext)

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}
