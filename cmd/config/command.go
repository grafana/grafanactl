package config

import (
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

type Options struct {
	ConfigFile string
	Context    string
}

func (opts *Options) BindFlags(flags *pflag.FlagSet) {
	flags.StringVar(&opts.ConfigFile, "config", "", "Path to the configuration file to use")
	flags.StringVar(&opts.Context, "context", "", "Name of the context to use")

	_ = cobra.MarkFlagFilename(flags, "config", "yaml", "yml")
}

func (opts *Options) LoadConfig() (config.Config, error) {
	overrides := []config.Override{
		// If Grafana-related env variables are set, use them to configure the
		// current context and Grafana config.
		func(cfg *config.Config) error {
			grafanaCfg := config.GrafanaConfig{}

			if err := env.Parse(&grafanaCfg); err != nil {
				return err
			}

			if !grafanaCfg.IsEmpty() {
				cfg.Contexts["default"] = &config.Context{
					Name:    "default",
					Grafana: &grafanaCfg,
				}
				cfg.CurrentContext = "default"
			}

			return nil
		},
	}

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

	return config.Load(opts.configSource(), overrides...)
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
		Short:   "Display the current configuration",
		Example: fmt.Sprintf("%s config view", path.Base(os.Args[0])),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := opts.Validate(); err != nil {
				return err
			}

			cfg, err := configOpts.LoadConfig()
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

			if err := opts.IO.Format(cfg, cmd.OutOrStdout()); err != nil {
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
		Short:   "Display the current context name",
		Long:    "Display the current context name.",
		Example: fmt.Sprintf("%s config current-context", path.Base(os.Args[0])),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := configOpts.LoadConfig()
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
		Aliases: []string{"use"},
		Short:   "Set the current context",
		Long:    "Set the current context and updates the configuration file.",
		Example: fmt.Sprintf("%s config use-context dev-instance", path.Base(os.Args[0])),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			if !cfg.HasContext(args[0]) {
				return config.ContextNotFound(args[0])
			}

			cfg.CurrentContext = args[0]

			if err := config.Write(configOpts.configSource(), cfg); err != nil {
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
		Short: "Set an single value in a configuration file",
		Long: `Set an single value in a configuration file

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.

PROPERTY_VALUE is the new value to set.`,
		Example: fmt.Sprintf(`
	# Set the "server" field on the "dev-instance" context to "https://grafana-dev.example"
	%[1]s config set contexts.dev-instance.grafana.server https://grafana-dev.example

	# Disable the validation of the server's SSL certificate in the "dev-instance" context
	%[1]s config set contexts.dev-instance.grafana.insecure-skip-tls-verify true`, path.Base(os.Args[0])),
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			if err := config.SetValue(&cfg, args[0], args[1]); err != nil {
				return err
			}

			return config.Write(configOpts.configSource(), cfg)
		},
	}

	return cmd
}

func unsetCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset PROPERTY_NAME",
		Short: "Unset an single value in a configuration file",
		Long: `Unset an single value in a configuration file.

PROPERTY_NAME is a dot-delimited reference to the value to unset. It can either represent a field or a map entry.`,
		Example: fmt.Sprintf(`
	# Unset the "foo" context
	%[1]s config unset contexts.foo

	# Unset the "insecure-skip-tls-verify" flag in the "dev-instance" context
	%[1]s config unset contexts.dev-instance.grafana.insecure-skip-tls-verify`, path.Base(os.Args[0])),
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			if err := config.UnsetValue(&cfg, args[0]); err != nil {
				return err
			}

			return config.Write(configOpts.configSource(), cfg)
		},
	}

	return cmd
}
