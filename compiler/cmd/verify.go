package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/log/v2"
	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/compiler"
	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/ir"
	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/schema"
	"github.com/spf13/cobra"
)

func validate(card *ir.Card) error {
	cardStr0, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("failed to marshal card: %v", err)
	}

	log.Infof("card str: %s", cardStr0)

	var data any
	err = json.Unmarshal(cardStr0, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal card: %v", err)
	}

	err = schema.Validate(data)
	if err != nil {
		return fmt.Errorf("failed to validate card: %v", err)
	}

	return nil
}

type VerifyResult struct {
	Source string `json:"source" yaml:"source"`
	Valid  bool   `json:"valid" yaml:"valid"`
	Error  string `json:"error,omitempty" yaml:"yaml,omitempty"`
}

func parseCardJson(data []byte) (*ir.Card, error) {
	var card ir.Card
	err := json.Unmarshal(data, &card)
	if err != nil {
		return nil, err
	}

	return &card, nil
}

func verify(sources []string) []VerifyResult {
	results := []VerifyResult{}
	appendValid := func(source string) {
		results = append(results, VerifyResult{
			Source: source,
			Valid:  true,
		})
	}
	appendInvalid := func(source string, err error) {
		results = append(results, VerifyResult{
			Source: source,
			Valid:  false,
			Error:  err.Error(),
		})
	}

	for _, source := range sources {
		data, err := os.ReadFile(source)
		if err != nil {
			appendInvalid(source, err)
			continue
		}

		filetype := strings.ToLower(filepath.Ext(source))
		log.Info("parsing json card", "filetype", filetype)

		var card *ir.Card
		switch filetype {
		case ".md":
			card, _, _, err = compiler.CompileMarkdown(string(data), compiler.Options{
				Where: "test",
				Provenance: compiler.Provenance{
					Commit:  "abc1234",
					BuiltAt: "2026-01-01T00:00:00Z",
				},
			})

		case ".json":
			card, err = parseCardJson(data)
			if err != nil {
				appendInvalid(source, err)
				continue
			}

		default:
			card, _, _, err = compiler.CompileMarkdown(string(data), compiler.Options{
				Where: "test",
				Provenance: compiler.Provenance{
					Commit:  "abc1234",
					BuiltAt: "2026-01-01T00:00:00Z",
				},
			})
		}

		err = validate(card)
		if err != nil {
			appendInvalid(source, err)
			continue
		}

		appendValid(source)
	}

	return results
}

func init() {
	c := &cobra.Command{
		Use:  "verify sources...",
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			results := verify(args)
			for _, result := range results {
				if !result.Valid {
					fmt.Println()
					fmt.Printf("error: %s invalid: %v\n", result.Source, result.Error)
					continue
				}

				fmt.Printf("- %s is valid\n", result.Source)
			}

			return nil
		},
	}
	RootCommand.AddCommand(c)
}
