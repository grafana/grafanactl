package root

import (
	"log/slog"
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/cmd/resources"
	"github.com/grafana/grafanactl/internal/logs"
	"github.com/spf13/cobra"
)

func Command(version string) *cobra.Command {
	noColors := false
	verbosity := 0

	rootCmd := &cobra.Command{
		Use:           path.Base(os.Args[0]),
		SilenceUsage:  true,
		SilenceErrors: true, // We want to print errors ourselves
		Version:       version,
		PersistentPreRun: func(cmd *cobra.Command, _ []string) {
			if noColors {
				color.NoColor = true // globally disables colorized output
			}

			logLevel := new(slog.LevelVar)
			logLevel.Set(slog.LevelWarn)
			logger := logging.NewSLogLogger(logs.NewHandler(os.Stderr, &logs.Options{
				Level: logLevel,
			}))

			// Multiplying the number of occurrences of the `-v` flag by 4 (gap between log levels in slog)
			// allows us to increase the logger's verbosity.
			logLevel.Set(logLevel.Level() - slog.Level(min(verbosity, 3)*4))

			cmd.SetContext(logging.Context(cmd.Context(), logger))
		},
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "grafanactl",
		},
	}

	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.SetIn(os.Stdin)

	rootCmd.AddCommand(config.Command())
	rootCmd.AddCommand(resources.Command())

	rootCmd.PersistentFlags().BoolVar(&noColors, "no-color", noColors, "Disable color output")
	rootCmd.PersistentFlags().CountVarP(&verbosity, "verbose", "v", "Verbose mode. Multiple -v options increase the verbosity (maximum: 3).")

	return rootCmd
}
