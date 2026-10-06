package cli

import "github.com/spf13/cobra"

func init() {
	compile := cobra.Command{
		Use: "compile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	RootCommand.AddCommand(&compile)
}
