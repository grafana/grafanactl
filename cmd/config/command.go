package config

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Apply a YAML dashboard",
		Run: func(_ *cobra.Command, _ []string) {
			//nolint:forbidigo
			fmt.Println("config subcommand")
		},
	}

	return cmd
}
