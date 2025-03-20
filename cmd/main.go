package main

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/fatih/color"
	"github.com/grafana/grafanactl/cmd/config"
	"github.com/grafana/grafanactl/internal/fail"
	"github.com/spf13/cobra"
)

var version = "SNAPSHOT"

func main() {
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
	}

	rootCmd.AddCommand(config.Command())

	rootCmd.PersistentFlags().BoolVar(&noColors, "no-color", noColors, "Disable color output")

	handleError(rootCmd.Execute())
}

func handleError(err error) {
	if err == nil {
		return
	}

	exitCode := 1

	detailedErr := fail.DetailedError{}
	if !errors.As(err, &detailedErr) {
		detailedErr = fail.DetailedError{
			Parent:  err,
			Summary: "Unexpected error",
		}
	}

	fmt.Fprintln(os.Stderr, detailedErr.Error())

	if detailedErr.ExitCode != nil {
		exitCode = *detailedErr.ExitCode
	}

	os.Exit(exitCode)
}
