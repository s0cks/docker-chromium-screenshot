// Package cards discovers and compiles cards: each is one directory under
// CARDS_DIR, self-contained — its own card.{md,json,yaml,yml,jsonnet}, its
// own logo.*, theme.css and meta.{json,yaml,yml,jsonnet}, plus an optional
// lib/ of jsonnet imports, so a card can be copied, committed, or handed to
// another repo's CI as a single unit. A directory named with a leading
// underscore (conventionally "_shared") is reserved for a library shared
// across cards (dossier.libsonnet, a repo-wide theme.css, shared meta) and
// is never itself listed as a card.
package cards

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/go-jsonnet"
	"gopkg.in/yaml.v2"
)

// SharedDirName is the reserved, non-card subdirectory of CARDS_DIR holding
// a jsonnet library, a repo-wide theme.css, and/or shared meta.
const SharedDirName = "_shared"

// LibDirName is the implicit per-card jsonnet import directory — always
// added to the VM's JPaths if it exists, no declaration needed.
const LibDirName = "lib"

var logoNames = []string{"logo.png", "logo.jpg", "logo.jpeg", "logo.svg", "logo.webp"}

// Kind is which of the five card.* extensions a card's source is.
type Kind string

const (
	KindMarkdown Kind = "markdown"
	KindJSON     Kind = "json"
	KindYAML     Kind = "yaml"
	KindJsonnet  Kind = "jsonnet"
)

// sourceCandidates is both the file-extension search order for Card.Source
// and (via its Kind) the dispatch table a caller uses to pick a compiler.
// Markdown is checked first: a card with both card.md and, say, a stray
// card.json is Markdown, not an error — same "first match wins" rule the
// original LayoutSource used across json/yaml/yml/jsonnet.
var sourceCandidates = []struct {
	ext  string
	kind Kind
}{
	{".md", KindMarkdown},
	{".json", KindJSON},
	{".yaml", KindYAML},
	{".yml", KindYAML},
	{".jsonnet", KindJsonnet},
}

// Card is one card directory: cards/<id>/.
type Card struct {
	ID  string
	Dir string
}

// List returns every card directory under dir — immediate subdirectories not
// named "_shared" or otherwise prefixed with "_" — sorted by ID. A missing
// dir yields an empty list, not an error.
func List(dir string) ([]Card, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var out []Card
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), "_") {
			continue
		}

		out = append(out, Card{ID: e.Name(), Dir: filepath.Join(dir, e.Name())})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Find locates the card named id directly under dir.
func Find(dir, id string) (Card, bool, error) {
	if id == "" || strings.Contains(id, string(filepath.Separator)) || strings.HasPrefix(id, "_") {
		return Card{}, false, nil
	}

	info, err := os.Stat(filepath.Join(dir, id))
	if os.IsNotExist(err) || (err == nil && !info.IsDir()) {
		return Card{}, false, nil
	}

	if err != nil {
		return Card{}, false, err
	}

	return Card{ID: id, Dir: filepath.Join(dir, id)}, true, nil
}

// SharedDir is dir's reserved _shared subdirectory (may not exist).
func SharedDir(dir string) string { return filepath.Join(dir, SharedDirName) }

// Source is one card's entry file: card.md, or card.{json,yaml,yml,jsonnet}.
type Source struct {
	Path string
	Kind Kind
}

// Source finds this card's entry file — card.md, card.json, card.yaml,
// card.yml or card.jsonnet, in that preference order if more than one is
// present (a card should really only have one, but authoring mistakes
// shouldn't be ambiguous about which file wins).
func (c Card) Source() (Source, bool) {
	for _, cand := range sourceCandidates {
		p := filepath.Join(c.Dir, "card"+cand.ext)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return Source{Path: p, Kind: cand.kind}, true
		}
	}

	return Source{}, false
}

// HasMarkdown reports whether this card's source is card.md — a thin
// convenience over Source() for callers that only care about that
// distinction (the Astro build's getStaticPaths, notably).
func (c Card) HasMarkdown() bool {
	src, ok := c.Source()
	return ok && src.Kind == KindMarkdown
}

// MetaSource finds this card's implicit meta.{json,yaml,yml,jsonnet} —
// extra metadata merged into the compiled card, see Orchestrator.
func (c Card) MetaSource() (Source, bool) {
	for _, cand := range sourceCandidates {
		if cand.kind == KindMarkdown {
			continue // meta.md makes no sense: meta is always structured data
		}

		p := filepath.Join(c.Dir, "meta"+cand.ext)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return Source{Path: p, Kind: cand.kind}, true
		}
	}

	return Source{}, false
}

// LibDir is this card's own implicit jsonnet import directory
// (cards/<id>/lib/), always added to the VM's JPaths if it exists.
func (c Card) LibDir() (string, bool) {
	p := filepath.Join(c.Dir, LibDirName)
	info, err := os.Stat(p)
	return p, err == nil && info.IsDir()
}

// LogoURL returns this card's own logo.{png,jpg,jpeg,svg,webp} as a
// /cards/<id>/<file> URL the SPA can use directly, if one exists.
func (c Card) LogoURL() (string, bool) {
	for _, name := range logoNames {
		if info, err := os.Stat(filepath.Join(c.Dir, name)); err == nil && !info.IsDir() {
			return "/cards/" + c.ID + "/" + name, true
		}
	}

	return "", false
}

// Compiler evaluates card.{json,yaml,yml,jsonnet} files. It is safe for
// concurrent use; the underlying jsonnet VM is not, so each Compile call
// gets its own.
type Compiler struct{}

// Compile reads src and returns its JSON. For .json it validates and
// re-marshals (so output is always compact-able and errors are caught
// early); for .yaml/.yml it converts; for .jsonnet it evaluates with
// jpaths searched for imports — typically the card's own lib/ (implicit,
// always included if present), then CARDS_DIR/_shared/ (see
// Orchestrator.jsonnetPaths for the full, documented search order).
func (c *Compiler) Compile(src Source, jpaths []string) (json.RawMessage, error) {
	switch src.Kind {
	case KindJSON:
		b, err := os.ReadFile(src.Path)
		if err != nil {
			return nil, err
		}

		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.Marshal(v)

	case KindYAML:
		b, err := os.ReadFile(src.Path)
		if err != nil {
			return nil, err
		}

		var raw map[any]any
		if err := yaml.Unmarshal(b, &raw); err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.Marshal(convert(raw))

	case KindJsonnet:
		vm := jsonnet.MakeVM()
		vm.Importer(&jsonnet.FileImporter{JPaths: jpaths})
		out, err := vm.EvaluateFile(src.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.RawMessage(out), nil

	default:
		return nil, fmt.Errorf("unsupported card source kind %q for %s", src.Kind, src.Path)
	}
}

// EvalJsonnetFile is a narrow adapter for meta.LoadExtra: compile a single
// .jsonnet file to a JSON string with no YAML/JSON fallback.
func (c *Compiler) EvalJsonnetFile(path string) (string, error) {
	vm := jsonnet.MakeVM()
	vm.Importer(&jsonnet.FileImporter{JPaths: []string{filepath.Dir(path)}})
	return vm.EvaluateFile(path)
}

func convert(v any) any {
	switch v := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[fmt.Sprint(k)] = convert(val)
		}

		return m

	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = convert(val)
		}

		return out

	default:
		return v
	}
}
