package config

import (
	"fmt"

	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Apply a YAML dashboard",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("config subcommand")
		},
	}

	return cmd
}
