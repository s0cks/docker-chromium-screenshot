// Package compiler turns a card.* source into the canonical ir.Card — this
// file is the Markdown path (card.md), built on goldmark for prose and a
// small hand-written structural scanner for the parts that aren't prose:
// frontmatter, @include snippets, fenced blocks, and +++ panel dividers.
//
// goldmark only ever sees genuine Markdown spans, never the structural
// markers: a full CommonMark parser run over a +++ or a fenced ```block
// line would misinterpret them (--- in particular is already meaningful
// CommonMark — a thematic break or a Setext heading underline — which is
// exactly why this split exists; with one slide per file there's no more
// --- divider to worry about, but +++ and fences still need to be pulled
// out before whatever's left is handed to goldmark).
package compiler

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v2"

	"github.com/s0cks/docker-chromium-screenshot/compiler/internal/ir"
)

// DefaultWordBudget is used when a card's frontmatter doesn't set budget.
const DefaultWordBudget = 200

// DefaultPanelDivider is the line that splits a card into panels. Override
// per-compile for `compile --panel-divider` (kept configurable in case +++
// ever collides with something else; see cmd-level flag wiring).
const DefaultPanelDivider = "+++"

var (
	fenceRE       = regexp.MustCompile("^(`{3,})\\s*([\\w-]*)\\s*(.*)$")
	headingRE     = regexp.MustCompile(`^#{1,2}\s+(.+?)\s*#*\s*$`)
	htmlTagRE     = regexp.MustCompile(`<[^>]+>`)
	wordRE        = regexp.MustCompile(`[\p{L}\p{N}][\p{L}\p{N}'’.,%$-]*`)
	attrRE        = regexp.MustCompile(`([\w-]+)(?:=("[^"]*"|'[^']*'|\S+))?`)
	frontmatterRE = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n?`)
	includeRE     = regexp.MustCompile(`(?m)^@include\s+([\w-]+)\s*$`)
)

var mdNode = ir.MDBlocks // callout, quote carry raw markdown; everything else carries decoded data

var gm = goldmark.New(goldmark.WithExtensions(extension.GFM))

// MD renders a block of Markdown to HTML (goldmark + GFM: tables, strikethrough, linkify, tasklist).
func MD(src string) (string, error) {
	var buf bytes.Buffer
	if err := gm.Convert([]byte(src), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// MDI renders a short Markdown span inline — no wrapping <p>, matching how
// a stat label or a finding's lead-in reads. goldmark always produces a
// block (at minimum one <p>); for the common case of a single paragraph,
// stripping that wrapper gives the same inline result the JS mdi()
// (marked's parseInline) produces. Genuinely block-level content (a list,
// a table) falls back to the full block render — there's no sensible
// "inline" form of those anyway.
func MDI(src string) (string, error) {
	html, err := MD(src)
	if err != nil {
		return "", err
	}

	html = strings.TrimSuffix(html, "\n")
	if strings.HasPrefix(html, "<p>") && strings.HasSuffix(html, "</p>") && strings.Count(html, "<p>") == 1 {
		return html[len("<p>") : len(html)-len("</p>")], nil
	}

	return html, nil
}

// ParseAttrs parses `tone=red mono w=half title="Two words"` into a map,
// matching src/lib/dossier.js's parseAttrs exactly (bare words are bool
// true; true/false literals coerce to bool; quoted strings are unquoted).
func ParseAttrs(s string) map[string]any {
	attrs := map[string]any{}
	for _, m := range attrRE.FindAllStringSubmatch(s, -1) {
		key, raw := m[1], m[2]
		switch {
		case raw == "":
			attrs[key] = true

		case strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, `'`):
			attrs[key] = raw[1 : len(raw)-1]

		case raw == "true":
			attrs[key] = true

		case raw == "false":
			attrs[key] = false

		default:
			attrs[key] = raw

		}
	}

	return attrs
}

// SplitFrontmatter pulls the --- delimited YAML block off the front of src.
// A missing frontmatter block yields an empty meta map, not an error.
func SplitFrontmatter(src string) (map[string]any, string, error) {
	m := frontmatterRE.FindStringSubmatchIndex(src)
	if m == nil {
		return map[string]any{}, src, nil
	}

	var meta map[string]any
	if err := yaml.Unmarshal([]byte(src[m[2]:m[3]]), &meta); err != nil {
		return nil, "", fmt.Errorf("frontmatter: invalid YAML: %w", err)
	}

	if meta == nil {
		meta = map[string]any{}
	}

	return normalizeYAMLMap(meta), src[m[1]:], nil
}

// ExpandIncludes replaces `@include name` lines with frontmatter `snippets`
// entries, up to 5 passes deep (a snippet whose own text contains another
// @include is allowed; a cycle just runs out of passes and leaves the
// unresolved line in place rather than looping forever).
func ExpandIncludes(body string, snippets map[string]any, where string) (string, error) {
	out := body
	for range 5 {
		hit := false
		var expandErr error
		out = includeRE.ReplaceAllStringFunc(out, func(line string) string {
			name := includeRE.FindStringSubmatch(line)[1]
			v, ok := snippets[name]
			if !ok {
				expandErr = fmt.Errorf(`%s: unknown snippet "%s"`, where, name)
				return line
			}

			hit = true
			return strings.TrimSpace(fmt.Sprint(v))
		})

		if expandErr != nil {
			return "", expandErr
		}

		if !hit {
			break
		}
	}

	return out, nil
}

type compileState struct {
	panels   []ir.Panel
	panel    int
	divider  string
	mdLines  []string
	words    int
	title    string
	hasTitle bool
	where    string
}

func (s *compileState) flush() error {
	text := strings.TrimSpace(strings.Join(s.mdLines, "\n"))
	s.mdLines = nil
	if text == "" {
		return nil
	}

	html, err := MD(text)
	if err != nil {
		return fmt.Errorf("%s: %w", s.where, err)
	}

	s.words += countWords(text)
	s.panels[s.panel].Nodes = append(s.panels[s.panel].Nodes, ir.Node{Kind: "md", HTML: html})
	return nil
}

func countWords(s string) int {
	return len(wordRE.FindAllString(htmlTagRE.ReplaceAllString(s, " "), -1))
}

// CompileMarkdownBody compiles the post-frontmatter body of a card.md into
// its slide: panel splitting on divider (+++ by default), fenced blocks,
// the first H1/H2 as the title, everything else as Markdown. Returns the
// slide, a total word count (for the caller's budget check — the budget
// itself lives in frontmatter, not here), and any warnings (currently just
// "no title").
func CompileMarkdownBody(body string, divider string, where string) (ir.Slide, int, []string, error) {
	if divider == "" {
		divider = DefaultPanelDivider
	}

	maxPanels := 4
	st := &compileState{panels: []ir.Panel{{}}, divider: divider, where: where}

	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		if f := fenceRE.FindStringSubmatch(line); f != nil {
			ticks, name, rest := f[1], f[2], f[3]
			isBlock := ir.BlockNames[name]
			j := i + 1
			var fenceBody []string
			for j < len(lines) {
				if c := fenceRE.FindStringSubmatch(lines[j]); c != nil && len(c[1]) >= len(ticks) && c[2] == "" && c[3] == "" {
					break
				}

				fenceBody = append(fenceBody, lines[j])
				j++
			}

			if !isBlock {
				// an ordinary code fence: leave it as Markdown, untouched
				st.mdLines = append(st.mdLines, lines[i:min(j+1, len(lines))]...)
				i = j
				continue
			}

			if err := st.flush(); err != nil {
				return ir.Slide{}, 0, nil, err
			}

			raw := strings.Join(fenceBody, "\n")
			ctx := fmt.Sprintf("%s, `%s` block at line %d", where, name, i+1)
			node := ir.Node{Kind: "block", Name: name, Attrs: ParseAttrs(rest)}
			if mdNode[name] {
				node.MD = strings.TrimSpace(raw)
				st.words += countWords(node.MD)
			} else {
				data, err := parseYAMLBody(raw, ctx)
				if err != nil {
					return ir.Slide{}, 0, nil, err
				}

				node.Data = data
				st.words += countWords(collectText(data))
			}

			st.panels[st.panel].Nodes = append(st.panels[st.panel].Nodes, node)
			i = j
			continue
		}

		if strings.TrimRight(line, " \t\r") == divider {
			if st.panel+1 >= maxPanels {
				return ir.Slide{}, 0, nil, fmt.Errorf("%s: more than %d panels (too many %q dividers)", where, maxPanels, divider)
			}

			if err := st.flush(); err != nil {
				return ir.Slide{}, 0, nil, err
			}

			st.panel++
			st.panels = append(st.panels, ir.Panel{})
			continue
		}

		if h := headingRE.FindStringSubmatch(line); h != nil && !st.hasTitle {
			st.title, st.hasTitle = h[1], true
			continue
		}

		st.mdLines = append(st.mdLines, line)
	}

	if err := st.flush(); err != nil {
		return ir.Slide{}, 0, nil, err
	}

	var warnings []string
	if !st.hasTitle {
		warnings = append(warnings, `no title (add a "## Title" line)`)
	}

	var title *string
	if st.hasTitle {
		title = &st.title
	}

	return ir.Slide{Title: title, Kicker: title, Panels: st.panels}, st.words, warnings, nil
}

func parseYAMLBody(raw, ctx string) (any, error) {
	var v any
	if err := yaml.Unmarshal([]byte(raw), &v); err != nil {
		return nil, fmt.Errorf("%s: invalid YAML: %w", ctx, err)
	}

	return normalizeYAML(v), nil
}

// collectText flattens any decoded YAML/JSON value into one string for
// word-counting a block's data (findings, stats labels, etc. all count
// toward a card's budget the same as prose does).
func collectText(v any) string {
	var out []string
	var walk func(any)
	walk = func(v any) {
		switch v := v.(type) {
		case string:
			out = append(out, v)

		case []any:
			for _, e := range v {
				walk(e)
			}

		case map[string]any:
			for _, e := range v {
				walk(e)
			}

		}
	}

	walk(v)
	return strings.Join(out, " ")
}

// normalizeYAML converts yaml.v2's map[interface{}]interface{} (and nested
// occurrences of it) into map[string]any, recursively, so the result
// round-trips through encoding/json.
func normalizeYAML(v any) any {
	switch v := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[fmt.Sprint(k)] = normalizeYAML(val)
		}

		return m

	case map[string]any:
		// already string-keyed (the top-level frontmatter map decodes this
		// way), but its values can still be the interface{}-keyed shape —
		// recurse into them too, not just into map[interface{}]interface{}.
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[k] = normalizeYAML(val)
		}

		return m

	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = normalizeYAML(val)
		}

		return out

	case int:
		return v

	default:
		return v

	}
}

func normalizeYAMLMap(m map[string]any) map[string]any {
	out, _ := normalizeYAML(m).(map[string]any)
	if out == nil {
		return map[string]any{}
	}

	return out
}

// SizePresets mirrors the preset names the schema's meta.size enum allows.
var SizePresets = map[string][2]int{
	"landscape": {1600, 1000},
	"wide":      {1600, 900},
	"og":        {1200, 630},
	"square":    {1080, 1080},
	"portrait":  {1080, 1350},
	"story":     {1080, 1920},
}

var sizeWxHRE = regexp.MustCompile(`^(\d{3,5})\s*x\s*(\d{3,5})$`)

// normalizeSize turns "1600x900" into [1600, 900], passes a recognized
// preset name through as-is, and errors on anything else — the schema's
// meta.size is either a known preset string or an explicit [w, h] pair,
// never an arbitrary string.
func normalizeSize(v any) (any, error) {
	s, ok := v.(string)
	if !ok {
		return v, nil // already an array from the source (jsonnet/JSON/YAML path, or hand-authored)
	}

	key := strings.ToLower(strings.TrimSpace(s))
	if _, ok := SizePresets[key]; ok {
		return key, nil
	}

	if m := sizeWxHRE.FindStringSubmatch(key); m != nil {
		w, _ := strconv.Atoi(m[1])
		h, _ := strconv.Atoi(m[2])
		return []any{w, h}, nil
	}

	presets := make([]string, 0, len(SizePresets))
	for k := range SizePresets {
		presets = append(presets, k)
	}

	return nil, fmt.Errorf(`invalid size %q (use WxH or one of: %s)`, s, strings.Join(presets, ", "))
}

// ResolveSize returns the actual pixel [width, height] for a meta.size
// value — a preset name or an explicit pair — for callers (the render
// pipeline) that need real numbers rather than the schema's own union
// shape.
func ResolveSize(v any) (int, int, error) {
	switch v := v.(type) {
	case string:
		if wh, ok := SizePresets[strings.ToLower(v)]; ok {
			return wh[0], wh[1], nil
		}

	case []any:
		if len(v) == 2 {
			w, wok := toInt(v[0])
			h, hok := toInt(v[1])
			if wok && hok {
				return w, h, nil
			}
		}
	}

	return 0, 0, fmt.Errorf("invalid size value: %v", v)
}

func toInt(v any) (int, bool) {
	switch v := v.(type) {
	case int:
		return v, true

	case float64:
		return int(v), true

	default:
		return 0, false

	}
}

// Provenance is supplied by the caller (the compile command or the build
// pipeline), not read from the card itself: which binary/commit/build
// produced this compile. Always attached to the result's meta — per
// product decision, provenance is no longer an opt-in frontmatter flag.
type Provenance struct {
	Commit  string
	BuiltAt string // RFC3339; caller's responsibility to format
}

// Options controls how CompileMarkdown treats the source beyond the file
// content itself.
type Options struct {
	Where      string // path, for error messages
	Divider    string // panel divider; "" uses DefaultPanelDivider
	Provenance Provenance
}

// requiredString reads a required frontmatter string field, erroring with
// its name if absent or not a string — used for the one truly required
// field (series); everything else in meta has a sensible default.
func requiredString(meta map[string]any, key, where string) (string, error) {
	v, ok := meta[key]
	if !ok {
		return "", fmt.Errorf("%s: frontmatter is missing required field %q", where, key)
	}

	s, ok := v.(string)
	if !ok || s == "" {
		return "", fmt.Errorf("%s: frontmatter field %q must be a non-empty string", where, key)
	}

	return s, nil
}

// Directives are frontmatter keys that steer compilation itself rather
// than describing the card (meta:, include:) — stripped out of both the
// card's Meta.Extra and never shown to a reader, consumed instead by the
// orchestrator that has filesystem access to act on them (CompileMarkdown
// itself only sees source text, no directory).
type Directives struct {
	MetaFile string // meta: — an extra meta.{json,yaml,yml,jsonnet} to merge in, beyond the implicit meta.*
	Include  string // include: — an extra jsonnet -J directory, beyond the implicit lib/ (jsonnet cards only; harmless no-op for Markdown)
}

// CompileMarkdown is the Markdown-path entry point: source bytes in, a
// validated ir.Card out (callers should still run the result through
// internal/schema.Validate — this function focuses on Markdown-specific
// concerns, not re-deriving the schema's own rules), plus the frontmatter's
// compilation directives for the caller to act on. warnings are non-fatal
// (e.g. over budget); err is fatal (e.g. missing series, malformed YAML,
// an unrecognized block name sent unknown data).
func CompileMarkdown(src string, opts Options) (*ir.Card, Directives, []string, error) {
	rawMeta, body, err := SplitFrontmatter(src)
	if err != nil {
		return nil, Directives{}, nil, fmt.Errorf("%s: %w", opts.Where, err)
	}

	snippets, _ := rawMeta["snippets"].(map[string]any)
	expanded, err := ExpandIncludes(body, snippets, opts.Where)
	if err != nil {
		return nil, Directives{}, nil, err
	}

	slide, words, slideWarnings, err := CompileMarkdownBody(expanded, opts.Divider, opts.Where)
	if err != nil {
		return nil, Directives{}, nil, err
	}

	series, err := requiredString(rawMeta, "series", opts.Where)
	if err != nil {
		return nil, Directives{}, nil, err
	}

	theme, _ := rawMeta["theme"].(string)
	if theme != "light" && theme != "dark" {
		theme = "light"
	}

	accent, _ := rawMeta["accent"].(string)
	if accent == "" {
		accent = "purple"
	}

	size, err := normalizeSize(firstNonNil(rawMeta["size"], "landscape"))
	if err != nil {
		return nil, Directives{}, nil, fmt.Errorf("%s: %w", opts.Where, err)
	}

	budget := DefaultWordBudget
	if b, ok := rawMeta["budget"]; ok {
		if n, ok := toInt(b); ok && n > 0 {
			budget = n
		}
	}

	if words > budget {
		slideWarnings = append(slideWarnings, fmt.Sprintf("%d words exceeds the budget of %d; split it or cut copy", words, budget))
	}

	meta := ir.Meta{
		Series: series,
		Theme:  theme,
		Accent: accent,
		Size:   size,
		Budget: budget,
		Provenance: &ir.Provenance{
			Compiler: "markdown",
			Commit:   opts.Provenance.Commit,
			BuiltAt:  opts.Provenance.BuiltAt,
		},
		Extra: map[string]any{},
	}
	if author, ok := rawMeta["author"].(string); ok {
		meta.Author = author
	}

	// Logo has three states frontmatter must distinguish: key absent (auto-
	// detect — meta.Logo left nil, the field omitted from JSON via
	// omitempty, which the caller's filesystem-aware auto-detection step
	// then fills in), key explicitly null (suppress — meta.LogoSet records
	// that the author made a choice so the caller must not override it),
	// or an explicit string (use it verbatim, also not overridden).
	if logoVal, present := rawMeta["logo"]; present {
		meta.LogoSet = true
		if s, ok := logoVal.(string); ok {
			meta.Logo = &s
		}
	}

	// Directives steer compilation (the caller's job, with filesystem
	// access this package doesn't have) rather than describe the card —
	// captured here, then stripped so they don't leak into Extra.
	var directives Directives
	if s, ok := rawMeta["meta"].(string); ok {
		directives.MetaFile = s
	}

	if s, ok := rawMeta["include"].(string); ok {
		directives.Include = s
	}

	for _, protected := range []string{"series", "theme", "accent", "size", "author", "logo", "budget", "snippets", "include", "meta"} {
		delete(rawMeta, protected)
	}

	maps.Copy(meta.Extra, rawMeta)
	return &ir.Card{Meta: meta, Slide: slide}, directives, slideWarnings, nil
}

func firstNonNil(v any, fallback any) any {
	if v == nil {
		return fallback
	}

	return v
}
