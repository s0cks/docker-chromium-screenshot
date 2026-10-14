// Package orchestrator is the filesystem-aware layer above the two
// compiler paths (internal/compiler for Markdown, internal/cards for
// JSON/YAML/jsonnet): given a card.Card, it finds the right source file,
// dispatches to the matching compiler, resolves everything that needs a
// directory listing to answer — the implicit meta.*, lib/ and logo.* a
// card doesn't have to declare, plus the frontmatter's own include: and
// meta: directives — merges it all by a fixed precedence, and validates
// the result against the schema before handing it back. One CompileCard
// call is the whole "turn this card directory into a validated Card"
// operation; every HTTP handler and CLI command should go through it
// rather than re-deriving any of this.
package orchestrator

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/s0cks/docker-chromium-screenshot/service/internal/cards"
	gocompiler "github.com/s0cks/docker-chromium-screenshot/service/internal/compiler"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/ir"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/schema"
)

// protectedMetaFields can never be set by a meta file (implicit or
// frontmatter-declared) — only the card's own source. A card's identity
// and layout decisions shouldn't silently change because a shared meta
// file elsewhere was edited.
var protectedMetaFields = map[string]bool{
	"series": true, "theme": true, "accent": true, "size": true,
}

// Options are the CLI/server-wide settings that apply to every card
// compiled through this orchestrator — as opposed to a per-card directive
// (Directives), which comes from the card's own frontmatter instead.
type Options struct {
	CardsDir     string   // CARDS_DIR root, for resolving _shared/
	PanelDivider string   // "" uses compiler.DefaultPanelDivider
	ExtraLibDirs []string // --lib: global jsonnet import dirs, layered under a card's own lib/ and include:
	Commit       string   // build provenance
	Now          func() time.Time
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// CompileCard turns one card directory into a validated ir.Card. warnings
// are non-fatal (e.g. a Markdown card over its word budget); a non-nil
// error means the card could not be compiled or failed schema validation.
func CompileCard(c cards.Card, opts Options) (*ir.Card, []string, error) {
	src, ok := c.Source()
	if !ok {
		return nil, nil, fmt.Errorf("%s: no card.{md,json,yaml,yml,jsonnet} found", c.ID)
	}

	builtAt := opts.now().UTC().Format(time.RFC3339)
	compiler := &cards.Compiler{}

	var (
		card       *ir.Card
		directives gocompiler.Directives
		warnings   []string
		err        error
	)

	switch src.Kind {
	case cards.KindMarkdown:
		raw, rerr := os.ReadFile(src.Path)
		if rerr != nil {
			return nil, nil, rerr
		}
		card, directives, warnings, err = gocompiler.CompileMarkdown(string(raw), gocompiler.Options{
			Where:      src.Path,
			Divider:    opts.PanelDivider,
			Provenance: gocompiler.Provenance{Commit: opts.Commit, BuiltAt: builtAt},
		})

	case cards.KindJSON, cards.KindYAML, cards.KindJsonnet:
		jpaths := jsonnetPaths(c, opts, "")
		rawJSON, cerr := compiler.Compile(src, jpaths)
		if cerr != nil {
			return nil, nil, fmt.Errorf("%s: %w", src.Path, cerr)
		}
		card = &ir.Card{}
		if uerr := json.Unmarshal(rawJSON, card); uerr != nil {
			return nil, nil, fmt.Errorf("%s: compiled output doesn't match the card shape: %w", src.Path, uerr)
		}
		// A JSON/YAML/jsonnet card's directives (include:/meta:) live at
		// the top level of the same object, outside the schema's own
		// "meta"/"slide" — peeking at raw, not the already-decoded card,
		// since ir.Card has no field for them (same reason Markdown's
		// directives don't come from the Meta struct either).
		var top struct {
			Include string `json:"include"`
			Meta    string `json:"meta"`
		}
		json.Unmarshal(rawJSON, &top) // best-effort; absent fields just stay ""
		directives = gocompiler.Directives{Include: top.Include, MetaFile: top.Meta}
		card.Meta.Provenance = &ir.Provenance{Compiler: string(src.Kind), Commit: opts.Commit, BuiltAt: builtAt}

	default:
		return nil, nil, fmt.Errorf("%s: unsupported source kind %q", src.Path, src.Kind)
	}

	if err != nil {
		return nil, nil, err
	}

	if err := applyMeta(c, card, directives, opts); err != nil {
		return nil, warnings, err
	}

	resolveLogo(c, card)

	b, err := json.Marshal(card)
	if err != nil {
		return nil, warnings, fmt.Errorf("%s: %w", src.Path, err)
	}

	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, warnings, fmt.Errorf("%s: %w", src.Path, err)
	}

	if err := schema.Validate(v); err != nil {
		return nil, warnings, fmt.Errorf("%s: schema validation failed: %w", src.Path, err)
	}

	return card, warnings, nil
}

// applyMeta merges the implicit meta.* file (if present) and the
// frontmatter-declared meta: file (if present) into card, in that order,
// with the card's own frontmatter always winning — "frontmatter is most
// important, then derive defaults from meta or env" — and
// protectedMetaFields never settable by either file.
func applyMeta(c cards.Card, card *ir.Card, directives gocompiler.Directives, opts Options) error {
	merged := map[string]any{}
	// directives.Include only helps here, never the card's own compile: a
	// card.jsonnet can't know its own include: until after it's already
	// compiled (it would need the very directory it's trying to declare,
	// to evaluate far enough to declare it) — a real structural limit of
	// jsonnet, not an oversight. meta.* and a declared meta: file compile
	// strictly after the card itself, so by the time either is reached,
	// directives.Include is already known and safe to use for both.
	paths := jsonnetPaths(c, opts, directives.Include)

	if implicit, ok := c.MetaSource(); ok {
		m, err := loadMetaFile(implicit, paths)
		if err != nil {
			return fmt.Errorf("%s: %w", implicit.Path, err)
		}

		maps.Copy(merged, m)
	}

	if directives.MetaFile != "" {
		p := filepath.Join(c.Dir, directives.MetaFile)
		kind, ok := kindForExt(filepath.Ext(p))
		if !ok {
			return fmt.Errorf("%s: meta: %q has an unrecognized extension", c.Dir, directives.MetaFile)
		}

		m, err := loadMetaFile(cards.Source{Path: p, Kind: kind}, paths)
		if err != nil {
			return fmt.Errorf("%s: %w", p, err)
		}

		maps.Copy(merged, m)
	}

	for k, v := range merged {
		if protectedMetaFields[k] {
			continue // a meta file never overrides the card's own identity/layout fields
		}

		switch k {
		case "author":
			if card.Meta.Author == "" {
				if s, ok := v.(string); ok {
					card.Meta.Author = s
				}
			}

		case "logo":
			if !card.Meta.LogoSet {
				if s, ok := v.(string); ok {
					card.Meta.Logo = &s
					card.Meta.LogoSet = true
				}
			}

		default:
			if card.Meta.Extra == nil {
				card.Meta.Extra = map[string]any{}
			}

			if _, already := card.Meta.Extra[k]; !already {
				card.Meta.Extra[k] = v
			}

		}
	}

	return nil
}

func loadMetaFile(src cards.Source, jpaths []string) (map[string]any, error) {
	compiler := &cards.Compiler{}
	raw, err := compiler.Compile(src, jpaths)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("expected a JSON object, got: %w", err)
	}

	return m, nil
}

func kindForExt(ext string) (cards.Kind, bool) {
	switch ext {
	case ".json":
		return cards.KindJSON, true

	case ".yaml", ".yml":
		return cards.KindYAML, true

	case ".jsonnet":
		return cards.KindJsonnet, true

	default:
		return "", false

	}
}

// jsonnetPaths is the full, documented jsonnet import search order: the
// card's own lib/ (implicit, always included if it exists) first, then
// any --lib dirs given on the command line (opts.ExtraLibDirs — "an extra
// set of libsonnet sources to load, extraneous to the ones in lib/"), then
// CARDS_DIR/_shared/ (the pre-existing shared-library convention, kept for
// back-compat). extraInclude, if non-empty, is a card's own include:
// directive, resolved relative to the card's directory and searched right
// after its lib/ — before the global --lib dirs, since it's more specific
// to this one card.
func jsonnetPaths(c cards.Card, opts Options, extraInclude string) []string {
	var paths []string
	if lib, ok := c.LibDir(); ok {
		paths = append(paths, lib)
	}

	if extraInclude != "" {
		paths = append(paths, filepath.Join(c.Dir, extraInclude))
	}

	paths = append(paths, opts.ExtraLibDirs...)
	if opts.CardsDir != "" {
		paths = append(paths, cards.SharedDir(opts.CardsDir))
	}

	return paths
}

// resolveLogo fills in card.Meta.Logo from the card's own logo.* if the
// card made no logo choice of its own (LogoSet false — neither an explicit
// string nor an explicit null). A meta file is deliberately not consulted
// here: logo resolution is purely "this card's own directory," the same
// rule card.md and card.jsonnet both already follow.
func resolveLogo(c cards.Card, card *ir.Card) {
	if card.Meta.LogoSet {
		return
	}

	if url, ok := c.LogoURL(); ok {
		card.Meta.Logo = &url
	}
}
