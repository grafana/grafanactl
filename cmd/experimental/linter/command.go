package linter

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "linter",
	}

	cmd.AddCommand(lintCmd())
	cmd.AddCommand(newCmd())
	cmd.AddCommand(rulesCmd())

	return cmd
}
