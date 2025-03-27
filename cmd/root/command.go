package root

import (
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/cmd/resources"
	"github.com/spf13/cobra"
)

func Command(version string) *cobra.Command {
	noColors := false

	rootCmd := &cobra.Command{
		Use:           path.Base(os.Args[0]),
		SilenceUsage:  true,
		SilenceErrors: true, // We want to print errors ourselves
		Version:       version,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if noColors {
				color.NoColor = true // globally disables colorized output
			}
		},
		Annotations: map[string]string{
			cobra.CommandDisplayNameAnnotation: "grafanactl",
		},
	}

	rootCmd.AddCommand(config.Command())
	rootCmd.AddCommand(resources.Command())

	rootCmd.PersistentFlags().BoolVar(&noColors, "no-color", noColors, "Disable color output")

	return rootCmd
}
