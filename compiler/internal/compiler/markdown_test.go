package compiler

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustCompile(t *testing.T, src string) (*CardResult, []string) {
	t.Helper()
	card, _, warnings, err := CompileMarkdown(src, Options{Where: "test", Provenance: Provenance{Commit: "abc1234", BuiltAt: "2026-01-01T00:00:00Z"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	return &CardResult{Card: card}, warnings
}

type CardResult struct{ Card any }

func TestMinimalCard(t *testing.T) {
	src := "---\nseries: Weekly Metrics\n---\n## Signups this week\n\nUp across every channel.\n"
	r, warnings := mustCompile(t, src)
	_ = r
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
}

func TestMissingSeriesIsAnError(t *testing.T) {
	_, _, _, err := CompileMarkdown("---\ntheme: dark\n---\n## T\nbody\n", Options{Where: "test"})
	if err == nil {
		t.Fatal("expected an error for missing series")
	}
}

func TestDefaultsAndOverrides(t *testing.T) {
	src := "---\nseries: S\ntheme: dark\naccent: blue\n---\n## T\nbody\n"
	card, _, _, err := CompileMarkdown(src, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if card.Meta.Theme != "dark" || card.Meta.Accent != "blue" {
		t.Fatalf("unexpected: %+v", card.Meta)
	}

	card2, _, _, _ := CompileMarkdown("---\nseries: S\n---\n## T\nbody\n", Options{Where: "test"})
	if card2.Meta.Theme != "light" || card2.Meta.Accent != "purple" {
		t.Fatalf("expected defaults light/purple, got %+v", card2.Meta)
	}
}

func TestSizeNormalization(t *testing.T) {
	card, _, _, err := CompileMarkdown("---\nseries: S\nsize: 1200x630\n---\n## T\nbody\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	got, ok := card.Meta.Size.([]any)
	if !ok || got[0] != 1200 || got[1] != 630 {
		t.Fatalf("expected [1200 630], got %#v", card.Meta.Size)
	}

	card2, _, _, err := CompileMarkdown("---\nseries: S\nsize: SQUARE\n---\n## T\nbody\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if card2.Meta.Size != "square" {
		t.Fatalf("expected preset passthrough lowercased, got %#v", card2.Meta.Size)
	}

	_, _, _, err = CompileMarkdown("---\nseries: S\nsize: nonsense\n---\n## T\nbody\n", Options{Where: "test"})
	if err == nil {
		t.Fatal("expected an error for an invalid size")
	}
}

func TestLogoThreeStates(t *testing.T) {
	// absent: no signal, orchestrator should auto-detect
	c1, _, _, _ := CompileMarkdown("---\nseries: S\n---\n## T\nx\n", Options{Where: "test"})
	if c1.Meta.LogoSet || c1.Meta.Logo != nil {
		t.Fatalf("expected no logo signal, got LogoSet=%v Logo=%v", c1.Meta.LogoSet, c1.Meta.Logo)
	}

	// explicit null: suppress, orchestrator must not override
	c2, _, _, _ := CompileMarkdown("---\nseries: S\nlogo: null\n---\n## T\nx\n", Options{Where: "test"})
	if !c2.Meta.LogoSet || c2.Meta.Logo != nil {
		t.Fatalf("expected LogoSet=true Logo=nil, got LogoSet=%v Logo=%v", c2.Meta.LogoSet, c2.Meta.Logo)
	}

	// explicit string: use verbatim
	c3, _, _, _ := CompileMarkdown("---\nseries: S\nlogo: /cards/x/custom.png\n---\n## T\nx\n", Options{Where: "test"})
	if !c3.Meta.LogoSet || c3.Meta.Logo == nil || *c3.Meta.Logo != "/cards/x/custom.png" {
		t.Fatalf("unexpected: LogoSet=%v Logo=%v", c3.Meta.LogoSet, c3.Meta.Logo)
	}
}

func TestPanelSplittingZeroToThreeDividers(t *testing.T) {
	for n, want := range map[int]int{0: 1, 1: 2, 2: 3, 3: 4} {
		body := "## T\n"
		for range n {
			body += "panel text\n\n+++\n\n"
		}

		body += "last panel\n"
		card, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test"})
		if err != nil {
			t.Fatalf("n=%d: %v", n, err)
		}

		if len(card.Slide.Panels) != want {
			t.Fatalf("n=%d dividers: got %d panels, want %d", n, len(card.Slide.Panels), want)
		}
	}
}

func TestFourDividersIsAnError(t *testing.T) {
	body := "## T\na\n+++\nb\n+++\nc\n+++\nd\n+++\ne\n"
	_, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test"})
	if err == nil {
		t.Fatal("expected an error for 4 dividers (5 panels)")
	}
}

func TestCustomDivider(t *testing.T) {
	body := "## T\na\n===\nb\n"
	card, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test", Divider: "==="})
	if err != nil {
		t.Fatal(err)
	}

	if len(card.Slide.Panels) != 2 {
		t.Fatalf("expected 2 panels with a custom divider, got %d", len(card.Slide.Panels))
	}
}

func TestFencedBlockProducesDataNode(t *testing.T) {
	body := "## T\n```stats cols=2\n- { value: \"1\", label: a }\n```\n"
	card, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	nodes := card.Slide.Panels[0].Nodes
	if len(nodes) != 1 || nodes[0].Kind != "block" || nodes[0].Name != "stats" {
		t.Fatalf("unexpected nodes: %+v", nodes)
	}

	if nodes[0].Attrs["cols"] != "2" {
		t.Fatalf("unexpected attrs: %+v", nodes[0].Attrs)
	}
}

func TestCalloutIsMDBlock(t *testing.T) {
	body := "## T\n```callout tone=accent\n**hi**\n```\n"
	card, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	n := card.Slide.Panels[0].Nodes[0]
	if n.Kind != "block" || n.Name != "callout" || n.MD != "**hi**" {
		t.Fatalf("unexpected: %+v", n)
	}
}

func TestUnregisteredFenceStaysOrdinaryMarkdown(t *testing.T) {
	body := "## T\n```js\nconst a = 1;\n```\n"
	card, _, _, err := CompileMarkdown("---\nseries: S\n---\n"+body, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	n := card.Slide.Panels[0].Nodes[0]
	if n.Kind != "md" || !strings.Contains(n.HTML, "<code") {
		t.Fatalf("expected an ordinary code-fenced md node, got %+v", n)
	}
}

func TestIncludeExpandsSnippets(t *testing.T) {
	src := "---\nseries: S\nsnippets:\n  note: Shared note\n---\n## T\n@include note\n"
	card, _, _, err := CompileMarkdown(src, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(card.Slide.Panels[0].Nodes[0].HTML, "Shared note") {
		t.Fatalf("expected the snippet expanded, got %+v", card.Slide.Panels[0].Nodes[0])
	}
}

func TestUnknownIncludeIsAnError(t *testing.T) {
	_, _, _, err := CompileMarkdown("---\nseries: S\n---\n## T\n@include nope\n", Options{Where: "test"})
	if err == nil {
		t.Fatal("expected an error for an unknown snippet")
	}
}

func TestWordBudgetWarning(t *testing.T) {
	long := strings.Repeat("word ", 40)
	_, _, warnings, err := CompileMarkdown("---\nseries: S\nbudget: 10\n---\n## T\n"+long+"\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 1 || !strings.Contains(warnings[0], "exceeds the budget of 10") {
		t.Fatalf("expected a budget warning, got %v", warnings)
	}
}

func TestMissingTitleWarning(t *testing.T) {
	_, _, warnings, err := CompileMarkdown("---\nseries: S\n---\njust a paragraph, no heading\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "no title") {
			found = true
		}
	}

	if !found {
		t.Fatalf("expected a no-title warning, got %v", warnings)
	}
}

func TestProtectedFieldsDoNotLeakIntoExtra(t *testing.T) {
	card, _, _, err := CompileMarkdown("---\nseries: S\norg: acme\n---\n## T\nx\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if card.Meta.Extra["org"] != "acme" {
		t.Fatalf("expected org to survive into Extra, got %+v", card.Meta.Extra)
	}

	if _, present := card.Meta.Extra["series"]; present {
		t.Fatal("series must not duplicate into Extra")
	}
}

func TestProvenanceAlwaysPresent(t *testing.T) {
	card, _, _, err := CompileMarkdown("---\nseries: S\n---\n## T\nx\n", Options{Where: "test", Provenance: Provenance{Commit: "deadbee", BuiltAt: "2026-02-02T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}

	if card.Meta.Provenance == nil || card.Meta.Provenance.Commit != "deadbee" || card.Meta.Provenance.Compiler != "markdown" {
		t.Fatalf("unexpected provenance: %+v", card.Meta.Provenance)
	}
}

func TestMarshalsToValidSchemaShape(t *testing.T) {
	src := `---
series: Weekly Metrics
author: Tazz
theme: dark
accent: blue
size: landscape
budget: 300
---

## Signups this week

Up across every channel.

` + "```stats cols=2\n- { value: \"1,204\", label: New signups, tone: ink }\n```" + `

+++

### Details

- a
- b
`
	card, _, warnings, err := CompileMarkdown(src, Options{Where: "test", Provenance: Provenance{Commit: "abc1234", BuiltAt: "2026-01-01T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}

	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %v", warnings)
	}

	b, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}

	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}

	if _, present := v["slides"]; present {
		t.Fatal("must marshal as singular 'slide', not 'slides'")
	}

	if v["slide"] == nil {
		t.Fatalf("expected a slide key: %s", b)
	}

	panels := v["slide"].(map[string]any)["panels"].([]any)
	if len(panels) != 2 {
		t.Fatalf("expected 2 panels, got %d: %s", len(panels), b)
	}
}

func TestDirectivesExtractedAndStrippedFromExtra(t *testing.T) {
	src := "---\nseries: S\ninclude: ../shared-lib\nmeta: overrides.yaml\norg: acme\n---\n## T\nx\n"
	card, directives, _, err := CompileMarkdown(src, Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if directives.Include != "../shared-lib" || directives.MetaFile != "overrides.yaml" {
		t.Fatalf("unexpected directives: %+v", directives)
	}

	if _, present := card.Meta.Extra["include"]; present {
		t.Fatal("include must not leak into Extra")
	}

	if _, present := card.Meta.Extra["meta"]; present {
		t.Fatal("meta must not leak into Extra")
	}

	if card.Meta.Extra["org"] != "acme" {
		t.Fatalf("ordinary extras should still pass through: %+v", card.Meta.Extra)
	}
}

func TestDirectivesEmptyWhenNotDeclared(t *testing.T) {
	_, directives, _, err := CompileMarkdown("---\nseries: S\n---\n## T\nx\n", Options{Where: "test"})
	if err != nil {
		t.Fatal(err)
	}

	if directives.Include != "" || directives.MetaFile != "" {
		t.Fatalf("expected empty directives, got %+v", directives)
	}
}
