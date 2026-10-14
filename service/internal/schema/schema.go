// Package schema embeds and compiles card.schema.json: the one JSON Schema
// (draft 2020-12) every compiled card — Markdown or JSON/YAML/jsonnet —
// must satisfy. See internal/ir for the matching Go structs and
// docs/ir-schema.md (once written) for the human-readable version.
package schema

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

//go:embed card.schema.json
var raw []byte

// Raw returns the embedded schema bytes as-is — e.g. for `compile schema`
// to print it, or for an external tool to validate against independently.
func Raw() []byte { return raw }

var compiled *jsonschema.Schema

func init() {
	c := jsonschema.NewCompiler()
	c.Draft = jsonschema.Draft2020
	if err := c.AddResource("card.schema.json", bytes.NewReader(raw)); err != nil {
		panic(fmt.Errorf("schema: embedded card.schema.json is invalid: %w", err))
	}

	s, err := c.Compile("card.schema.json")
	if err != nil {
		panic(fmt.Errorf("schema: embedded card.schema.json failed to compile: %w", err))
	}

	compiled = s
}

// Validate checks cardJSON (an already-decoded `any`, typically from
// json.Unmarshal of a compiled card) against the schema. A nil error means
// valid; otherwise the returned error is a *jsonschema.ValidationError,
// whose .BasicOutput() gives a structured, path-annotated error list
// (what `compile --validate` prints).
func Validate(cardJSON any) error {
	return compiled.Validate(cardJSON)
}
