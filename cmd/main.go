package main

import (
	"os"
	"path"

	"github.com/grafana/grafanactl/cmd/config"
	"github.com/spf13/cobra"
)

var version = "SNAPSHOT"

func main() {
	rootCmd := &cobra.Command{
		Use:          path.Base(os.Args[0]),
		SilenceUsage: true,
		Version:      version,
	}

	rootCmd.AddCommand(config.Command())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
