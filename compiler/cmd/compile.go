package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"charm.land/log/v2"
	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/compiler"
	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use: "compile",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Info("compiling....")

			file := args[0]
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}

			card, _, _, err := compiler.CompileMarkdown(string(data), compiler.Options{
				Where: "test",
				Provenance: compiler.Provenance{
					Commit:  "abc1234",
					BuiltAt: "2026-01-01T00:00:00Z",
				},
			})

			encoded, err := json.MarshalIndent(card, "", "  ")
			if err != nil {
				return err
			}

			fmt.Println(string(encoded))
			return nil
		},
	}
	RootCommand.AddCommand(c)
}
