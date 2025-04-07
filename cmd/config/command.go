package config

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/caarlos0/env/v11"
	"github.com/grafana/grafanactl/cmd/io"
	"github.com/grafana/grafanactl/internal/config"
	"github.com/grafana/grafanactl/internal/secrets"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

//nolint:gochecknoglobals
var binaryName = path.Base(os.Args[0])

type Options struct {
	ConfigFile string
	Context    string
}

func (opts *Options) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.ConfigFile, "config", "", "Path to the configuration file to use")
	flags.StringVar(&opts.Context, "context", "", "Name of the context to use")

	_ = cobra.MarkFlagFilename(flags, "config", "yaml", "yml")
}

// loadConfigTolerant loads the configuration file (default, or explicitly set via flags)
// and returns it without validation.
// This function should only be used by config-related commands, to allow the
// user to iterate on the configuration until it becomes valid.
func (opts *Options) loadConfigTolerant(ctx context.Context, extraOverrides ...config.Override) (config.Config, error) {
	overrides := append([]config.Override{
		// If Grafana-related env variables are set, use them to configure the
		// current context and Grafana config.
		func(cfg *config.Config) error {
			grafanaCfg := config.GrafanaConfig{}

			if err := env.Parse(&grafanaCfg); err != nil {
				return err
			}

			if !grafanaCfg.IsEmpty() {
				cfg.SetContext(config.DefaultContextName, true, config.Context{
					Name:    config.DefaultContextName,
					Grafana: &grafanaCfg,
				})
			}

			return nil
		},
	}, extraOverrides...)

	// The current context is being overridden by a flag
	if opts.Context != "" {
		overrides = append(overrides, func(cfg *config.Config) error {
			if !cfg.HasContext(opts.Context) {
				return config.ContextNotFound(opts.Context)
			}

			cfg.CurrentContext = opts.Context
			return nil
		})
	}

	return config.Load(ctx, opts.configSource(), overrides...)
}

// LoadConfig loads the configuration file (default, or explicitly set via flags)
// and validates it.
func (opts *Options) LoadConfig(ctx context.Context) (config.Config, error) {
	validator := func(cfg *config.Config) error {
		// Ensure that the current context actually exists.
		if !cfg.HasContext(cfg.CurrentContext) {
			return config.ContextNotFound(cfg.CurrentContext)
		}

		return cfg.GetCurrentContext().Validate()
	}

	return opts.loadConfigTolerant(ctx, validator)
}

func (opts *Options) configSource() config.Source {
	if opts.ConfigFile != "" {
		return config.ExplicitConfigFile(opts.ConfigFile)
	}

	return config.StandardLocation()
}

func Command() *cobra.Command {
	configOpts := &Options{}

	cmd := &cobra.Command{
		Use:   "config",
		Short: "View or manipulate configuration settings",
		Long: fmt.Sprintf(`View or manipulate configuration settings.

The configuration file to load is chosen as follows:

1. If the --config flag is set, then that file will be loaded. No other location will be considered.
2. If the $XDG_CONFIG_HOME environment variable is set, then it will be used: $XDG_CONFIG_HOME/%[1]s/%[2]s
   Example: /home/user/.config/%[1]s/%[2]s
3. If the $HOME environment variable is set, then it will be used: $HOME/.config/%[1]s/%[2]s
   Example: /home/user/.config/%[1]s/%[2]s
4. If the $XDG_CONFIG_DIRS environment variable is set, then it will be used: $XDG_CONFIG_DIRS/%[1]s/%[2]s
   Example: /etc/xdg/%[1]s/%[2]s
`, config.StandardConfigFolder, config.StandardConfigFileName),
	}

	configOpts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(currentContextCmd(configOpts))
	cmd.AddCommand(setCmd(configOpts))
	cmd.AddCommand(unsetCmd(configOpts))
	cmd.AddCommand(useContextCmd(configOpts))
	cmd.AddCommand(viewCmd(configOpts))

	return cmd
}

type viewOpts struct {
	IO io.Options

	Minify bool
	Raw    bool
}

func (opts *viewOpts) BindFlags(flags *pflag.FlagSet) {
	opts.IO.BindFlags(flags)

	flags.BoolVar(&opts.Minify, "minify", opts.Minify, "Remove all information not used by current-context from the output")
	flags.BoolVar(&opts.Raw, "raw", opts.Raw, "Display sensitive information")
}

func (opts *viewOpts) Validate() error {
	if err := opts.IO.Validate(); err != nil {
		return err
	}

	return nil
}

func viewCmd(configOpts *Options) *cobra.Command {
	opts := &viewOpts{}

	cmd := &cobra.Command{
		Use:     "view",
		Args:    cobra.NoArgs,
		Short:   "Display the current configuration",
		Example: "\n\t" + binaryName + " config view",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.loadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			if opts.Minify {
				cfg, err = config.Minify(cfg)
				if err != nil {
					return err
				}
			}

			if !opts.Raw {
				if err := secrets.Redact(&cfg); err != nil {
					return fmt.Errorf("could not redact secrets from configuration: %w", err)
				}
			}

			if err := opts.IO.Format(cmd.OutOrStdout(), cfg); err != nil {
				return err
			}

			return nil
		},
	}

	opts.BindFlags(cmd.Flags())

	return cmd
}

func currentContextCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "current-context",
		Args:    cobra.NoArgs,
		Short:   "Display the current context name",
		Long:    "Display the current context name.",
		Example: "\n\t" + binaryName + " config current-context",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.loadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			cmd.Println(cfg.CurrentContext)

			return nil
		},
	}

	return cmd
}

func useContextCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "use-context CONTEXT_NAME",
		Args:    cobra.ExactArgs(1),
		Aliases: []string{"use"},
		Short:   "Set the current context",
		Long:    "Set the current context and updates the configuration file.",
		Example: "\n\t" + binaryName + " config use-context dev-instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.loadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			if !cfg.HasContext(args[0]) {
				return config.ContextNotFound(args[0])
			}

			cfg.CurrentContext = args[0]

			if err := config.Write(cmd.Context(), configOpts.configSource(), cfg); err != nil {
				return err
			}

			io.Success(cmd.OutOrStdout(), "Context set to \"%s\"", cfg.CurrentContext)
			return nil
		},
	}

	return cmd
}

func setCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set PROPERTY_NAME PROPERTY_VALUE",
		Args:  cobra.ExactArgs(2),
		Short: "Set an single value in a configuration file",
		Long: `Set an single value in a configuration file

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.

PROPERTY_VALUE is the new value to set.`,
		Example: fmt.Sprintf(`
	# Set the "server" field on the "dev-instance" context to "https://grafana-dev.example"
	%[1]s config set contexts.dev-instance.grafana.server https://grafana-dev.example

	# Disable the validation of the server's SSL certificate in the "dev-instance" context
	%[1]s config set contexts.dev-instance.grafana.insecure-skip-tls-verify true`, binaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.loadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			if err := config.SetValue(&cfg, args[0], args[1]); err != nil {
				return err
			}

			return config.Write(cmd.Context(), configOpts.configSource(), cfg)
		},
	}

	return cmd
}

func unsetCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset PROPERTY_NAME",
		Args:  cobra.ExactArgs(1),
		Short: "Unset an single value in a configuration file",
		Long: `Unset an single value in a configuration file.

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.`,
		Example: fmt.Sprintf(`
	# Unset the "foo" context
	%[1]s config unset contexts.foo

	# Unset the "insecure-skip-tls-verify" flag in the "dev-instance" context
	%[1]s config unset contexts.dev-instance.grafana.insecure-skip-tls-verify`, binaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.loadConfigTolerant(cmd.Context())
			if err != nil {
				return err
			}

			if err := config.UnsetValue(&cfg, args[0]); err != nil {
				return err
			}

			return config.Write(cmd.Context(), configOpts.configSource(), cfg)
		},
	}

	return cmd
}
