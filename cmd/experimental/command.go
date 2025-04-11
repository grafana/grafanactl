package experimental

import (
	"github.com/grafana/grafanactl/cmd/experimental/linter"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experimental",
		Short: "Collection of experimental commands",
		Long:  `Collection of experimental commands.`,
	}

	cmd.AddCommand(linter.Command())

	return cmd
}
