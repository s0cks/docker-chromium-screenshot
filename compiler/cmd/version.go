package cmd

import (
	"fmt"

	"github.com/s0cks/docker-chromium-screenshot/compiler/internal"
	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(internal.GetVersion())
			return nil
		},
	}
	RootCommand.AddCommand(c)
}
