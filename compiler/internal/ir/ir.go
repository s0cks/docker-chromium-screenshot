// Package ir defines the canonical card representation: the one JSON shape
// both the Markdown compiler (internal/compiler, goldmark-based) and the
// JSON/YAML/jsonnet compiler (internal/cards) produce, validated against
// schema/card.schema.json before anything downstream (the Astro build, the
// live SPA, `compile --validate`) touches it. One card.* file compiles to
// exactly one Card: one image, one to four panels.
package ir

import "encoding/json"

type Card struct {
	Meta  Meta  `json:"meta"`
	Slide Slide `json:"slide"`
}

type Meta struct {
	Series string  `json:"series"`
	Author string  `json:"author,omitempty"`
	Theme  string  `json:"theme"`
	Accent string  `json:"accent"`
	Size   any     `json:"size"` // string preset, or [width, height]
	Logo   *string `json:"logo,omitempty"`
	Title  string  `json:"title,omitempty"`
	// LogoSet distinguishes "frontmatter didn't mention logo" (false — the
	// orchestrator should auto-detect one) from "frontmatter set logo,
	// including explicitly to null" (true — auto-detection must not
	// override it, even though Logo may itself still be nil). Not part of
	// the public schema; a compile-time signal only.
	LogoSet    bool        `json:"-"`
	Budget     int         `json:"budget,omitempty"`
	Provenance *Provenance `json:"provenance,omitempty"`

	// Extra holds any additional meta fields a meta.* file or frontmatter
	// contributed beyond the schema's named properties — merged back in at
	// MarshalJSON time so a card can carry arbitrary operator metadata
	// (org, owner, ...) without the schema having to enumerate it.
	Extra map[string]any `json:"-"`
}

type Provenance struct {
	Compiler string `json:"compiler"` // markdown | json | yaml | jsonnet
	Commit   string `json:"commit"`
	BuiltAt  string `json:"builtAt"` // RFC3339
}

type Slide struct {
	Kicker *string `json:"kicker,omitempty"`
	Panels []Panel `json:"panels"`
}

type Panel struct {
	Nodes []Node `json:"nodes"`
}

// Node is either a Markdown span ({kind:"md", html}) or a block
// ({kind:"block", name, attrs?, data?, md?}). Exactly one of Data/MD is set
// for a block node, matching BLOCKS in src/lib/dossier.js: callout and
// quote carry MD (a Markdown-bodied fence); everything else carries Data
// (a YAML/JSON-bodied fence, already decoded).
type Node struct {
	Kind  string         `json:"kind"` // "md" | "block"
	HTML  string         `json:"html,omitempty"`
	Name  string         `json:"name,omitempty"`
	Attrs map[string]any `json:"attrs,omitempty"`
	Data  any            `json:"data,omitempty"`
	MD    string         `json:"md,omitempty"`
}

// BlockNames is the registered block vocabulary — kept as the single
// source of truth for both schema generation and compiler-side validation
// (an unrecognized block name is a compile error, not a silently-dropped
// node).
var BlockNames = map[string]bool{
	"callout":  true,
	"quote":    true,
	"stats":    true,
	"findings": true,
	"threads":  true,
	"timeline": true,
	"chart":    true,
	"profile":  true,
	"sources":  true,
	"qr":       true,
	"image":    true,
}

// MDBlocks carries a Markdown-bodied fence (md field); every other block
// name carries a YAML/JSON-bodied one (data field).
var MDBlocks = map[string]bool{"callout": true, "quote": true}

// namedMetaFields are Meta's schema-recognized keys — anything else in a
// decoded object goes to Extra instead. Kept as a slice (not derived via
// reflection) so it reads as an explicit, auditable list next to the
// struct it mirrors.
var namedMetaFields = []string{"series", "author", "theme", "accent", "size", "logo", "budget", "provenance"}

// UnmarshalJSON is the counterpart to MarshalJSON: named fields decode
// normally, everything else in the object becomes Extra. Used when a
// JSON/YAML/jsonnet card writes its meta object directly (as opposed to
// Markdown frontmatter, which the compiler package parses into a Meta by
// hand) — the card author's own field choices are the source of truth
// here, this just separates "the schema knows this one" from "pass it
// through."
func (m *Meta) UnmarshalJSON(b []byte) error {
	type alias Meta
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}

	*m = Meta(a)
	var all map[string]json.RawMessage
	if err := json.Unmarshal(b, &all); err != nil {
		return err
	}

	// logo's presence (even as explicit null) is itself meaningful — see
	// Meta.LogoSet — which a plain struct decode into *string loses (both
	// "absent" and "null" decode to a nil pointer). Check before deleting
	// named fields from the map below.
	if _, ok := all["logo"]; ok {
		m.LogoSet = true
	}

	for _, f := range namedMetaFields {
		delete(all, f)
	}

	if len(all) == 0 {
		m.Extra = nil
		return nil
	}

	m.Extra = make(map[string]any, len(all))
	for k, raw := range all {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return err
		}

		m.Extra[k] = v
	}

	return nil
}

func (m Meta) MarshalJSON() ([]byte, error) {
	type alias Meta
	base, err := json.Marshal(alias(m))
	if err != nil {
		return nil, err
	}

	if len(m.Extra) == 0 {
		return base, nil
	}

	var merged map[string]any
	if err := json.Unmarshal(base, &merged); err != nil {
		return nil, err
	}

	for k, v := range m.Extra {
		if _, taken := merged[k]; !taken {
			merged[k] = v
		}
	}

	return json.Marshal(merged)
}
