package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/s0cks/docker-chromium-screenshot/service/internal/compiler"
	"github.com/spf13/cobra"
)

func init() {
	compile := &cobra.Command{
		Use:  "compile",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
	RootCommand.AddCommand(compile)
}
