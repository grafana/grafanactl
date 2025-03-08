package config

import (
	"fmt"

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
		Use:   "view",
		Short: "Display the current configuration settings",
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
		Use:   "current-context",
		Short: "Display the current context name",
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
		Use:   "use-context CONTEXT_NAME",
		Short: "Set the current context",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			cfg, err := configOpts.LoadConfig()
			if err != nil {
				return err
			}

			_, found := cfg.Contexts[args[0]]
			if !found {
				return fmt.Errorf("no context found with name %q", args[0])
			}

			cfg.CurrentContext = args[0]

			return config.Write(configOpts.configSource(), cfg)
		},
	}

	return cmd
}

func setCmd(configOpts *Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set PROPERTY_NAME PROPERTY_VALUE",
		Short: "Set an individual value in a configuration file",
		Args:  cobra.ExactArgs(2),
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
		Short: "Unset an individual value in a configuration file",
		Args:  cobra.ExactArgs(1),
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
