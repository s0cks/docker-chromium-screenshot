package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/schema"
)

// TestRealCardsCompileAndValidate ports actual panel content from the
// repo's real cards/*/card.md files (single-slide excerpts, since those
// files predate the one-card-one-slide model and are multi-slide) and
// proves the compiler produces schema-valid output for genuine content,
// not just synthetic test fixtures.
func TestRealCardsCompileAndValidate(t *testing.T) {
	fixtures := []string{"testdata/documents_show_slide.md", "testdata/showcase_slide.md"}
	for _, f := range fixtures {
		t.Run(f, func(t *testing.T) {
			src, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			card, _, warnings, err := CompileMarkdown(string(src), Options{
				Where:      f,
				Provenance: Provenance{Commit: "abc1234", BuiltAt: "2026-01-01T00:00:00Z"},
			})
			if err != nil {
				t.Fatalf("compile: %v", err)
			}

			t.Logf("warnings: %v", warnings)

			b, err := json.Marshal(card)
			if err != nil {
				t.Fatal(err)
			}

			var v any
			if err := json.Unmarshal(b, &v); err != nil {
				t.Fatal(err)
			}

			if err := schema.Validate(v); err != nil {
				t.Fatalf("schema validation failed: %v\n\ncompiled JSON:\n%s", err, b)
			}
		})
	}
}

func TestMain_fixtureDirExists(t *testing.T) {
	if _, err := os.Stat(filepath.Join("testdata")); err != nil {
		t.Skip("no testdata dir")
	}
}
