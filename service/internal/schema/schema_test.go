package schema

import (
	"encoding/json"
	"testing"
)

func decode(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatal(err)
	}

	return v
}

func TestValidMinimalCard(t *testing.T) {
	v := decode(t, `{
		"meta": {"series": "S", "theme": "light", "accent": "purple", "size": "landscape"},
		"slide": {"panels": [{"nodes": []}]}
	}`)
	if err := Validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidFullCard(t *testing.T) {
	v := decode(t, `{
		"meta": {
			"series": "S", "author": "A", "theme": "dark", "accent": "blue", "size": [1600, 1000],
			"logo": "/cards/x/logo.png", "budget": 200,
			"provenance": {"compiler": "markdown", "commit": "abc1234", "builtAt": "2026-01-01T00:00:00Z"},
			"org": "acme"
		},
		"slide": {
			"title": "T", "kicker": "K",
			"panels": [
				{"nodes": [{"kind": "md", "html": "<p>hi</p>"}]},
				{"nodes": [{"kind": "block", "name": "stats", "attrs": {"cols": 2}, "data": [{"value": "1"}]}]}
			]
		}
	}`)
	if err := Validate(v); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestRejectsMissingRequiredMeta(t *testing.T) {
	v := decode(t, `{"meta": {"theme": "light", "accent": "purple", "size": "landscape"}, "slide": {"panels": [{"nodes": []}]}}`)
	if err := Validate(v); err == nil {
		t.Fatal("expected an error for missing series")
	}
}

func TestRejectsUnknownBlockName(t *testing.T) {
	v := decode(t, `{
		"meta": {"series": "S", "theme": "light", "accent": "purple", "size": "landscape"},
		"slide": {"panels": [{"nodes": [{"kind": "block", "name": "not-a-real-block"}]}]}
	}`)
	if err := Validate(v); err == nil {
		t.Fatal("expected an error for an unregistered block name")
	}
}

func TestRejectsFivePanels(t *testing.T) {
	panels := `[{"nodes":[]},{"nodes":[]},{"nodes":[]},{"nodes":[]},{"nodes":[]}]`
	v := decode(t, `{"meta": {"series": "S", "theme": "light", "accent": "purple", "size": "landscape"}, "slide": {"panels": `+panels+`}}`)
	if err := Validate(v); err == nil {
		t.Fatal("expected an error for more than 4 panels")
	}
}

func TestRejectsUnknownTopLevelField(t *testing.T) {
	v := decode(t, `{
		"meta": {"series": "S", "theme": "light", "accent": "purple", "size": "landscape"},
		"slide": {"panels": [{"nodes": []}]},
		"slides": []
	}`)
	if err := Validate(v); err == nil {
		t.Fatal("expected an error for a stray 'slides' (plural) field — the schema requires singular 'slide'")
	}
}

func TestRejectsBadTheme(t *testing.T) {
	v := decode(t, `{"meta": {"series": "S", "theme": "sepia", "accent": "purple", "size": "landscape"}, "slide": {"panels": [{"nodes": []}]}}`)
	if err := Validate(v); err == nil {
		t.Fatal("expected an error for an invalid theme enum value")
	}
}

func TestRawReturnsEmbeddedBytes(t *testing.T) {
	if len(Raw()) == 0 {
		t.Fatal("expected non-empty embedded schema")
	}
}
