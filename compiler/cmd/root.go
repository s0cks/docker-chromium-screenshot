package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"charm.land/log/v2"
	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/compiler"
	"github.com/spf13/cobra"
)

func getSource(args []string) ([]byte, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		return io.ReadAll(os.Stdin)
	}

	return os.ReadFile(args[0])
}

var RootCommand = &cobra.Command{
	Use:  "compiler",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := getSource(args)
		if err != nil {
			return err
		}

		log.Info("compiling....")
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

func Execute(ctx context.Context) error {
	return RootCommand.ExecuteContext(ctx)
}
