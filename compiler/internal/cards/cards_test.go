package cards

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListSkipsUnderscoreDirsAndFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b", "card.md"), "# B")
	writeFile(t, filepath.Join(dir, "a", "card.json"), "{}")
	writeFile(t, filepath.Join(dir, "_shared", "dossier.libsonnet"), "{}")
	writeFile(t, filepath.Join(dir, "not-a-dir.md"), "stray file, not a card")

	list, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Fatalf("unexpected: %+v", list)
	}
}

func TestListMissingDir(t *testing.T) {
	list, err := List(filepath.Join(t.TempDir(), "nope"))
	if err != nil || list != nil {
		t.Fatalf("want (nil, nil) for a missing dir, got (%v, %v)", list, err)
	}
}

func TestFindRejectsTraversalAndUnderscore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a", "card.md"), "# A")

	if _, ok, _ := Find(dir, "a"); !ok {
		t.Fatal("expected to find card a")
	}

	if _, ok, _ := Find(dir, "_shared"); ok {
		t.Fatal("_shared must not resolve as a card")
	}

	if _, ok, _ := Find(dir, "../a"); ok {
		t.Fatal("path traversal must not resolve")
	}

	if _, ok, _ := Find(dir, "nope"); ok {
		t.Fatal("missing card should not be found")
	}
}

func TestSourcePrefersMarkdownThenJSONThenYAMLThenJsonnet(t *testing.T) {
	dir := t.TempDir()
	card := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	writeFile(t, filepath.Join(card.Dir, "card.jsonnet"), "{}")
	writeFile(t, filepath.Join(card.Dir, "card.yaml"), "{}")
	src, ok := card.Source()
	if !ok || src.Kind != KindYAML {
		t.Fatalf("want yaml (no json/md present), got %+v ok=%v", src, ok)
	}

	writeFile(t, filepath.Join(card.Dir, "card.json"), "{}")
	src, ok = card.Source()
	if !ok || src.Kind != KindJSON {
		t.Fatalf("want json to win over yaml/jsonnet, got %+v ok=%v", src, ok)
	}

	writeFile(t, filepath.Join(card.Dir, "card.md"), "# A")
	src, ok = card.Source()
	if !ok || src.Kind != KindMarkdown {
		t.Fatalf("want markdown to win over everything else, got %+v ok=%v", src, ok)
	}
}

func TestHasMarkdown(t *testing.T) {
	dir := t.TempDir()
	md := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	writeFile(t, filepath.Join(md.Dir, "card.md"), "# A")
	if !md.HasMarkdown() {
		t.Fatal("expected HasMarkdown true")
	}

	js := Card{ID: "b", Dir: filepath.Join(dir, "b")}
	writeFile(t, filepath.Join(js.Dir, "card.json"), "{}")
	if js.HasMarkdown() {
		t.Fatal("expected HasMarkdown false")
	}
}

func TestMetaSourceSkipsMarkdownAndPrefersJSON(t *testing.T) {
	dir := t.TempDir()
	card := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	if _, ok := card.MetaSource(); ok {
		t.Fatal("no meta file yet")
	}

	writeFile(t, filepath.Join(card.Dir, "meta.yaml"), "owner: a")
	src, ok := card.MetaSource()
	if !ok || src.Kind != KindYAML {
		t.Fatalf("unexpected: %+v %v", src, ok)
	}

	writeFile(t, filepath.Join(card.Dir, "meta.json"), "{}")
	src, ok = card.MetaSource()
	if !ok || src.Kind != KindJSON {
		t.Fatalf("want json to win, got %+v %v", src, ok)
	}
}

func TestLibDir(t *testing.T) {
	dir := t.TempDir()
	card := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	os.MkdirAll(card.Dir, 0o755)
	if _, ok := card.LibDir(); ok {
		t.Fatal("no lib/ yet")
	}

	os.MkdirAll(filepath.Join(card.Dir, "lib"), 0o755)
	p, ok := card.LibDir()
	if !ok || p != filepath.Join(card.Dir, "lib") {
		t.Fatalf("unexpected: %q %v", p, ok)
	}
}

func TestLogoURL(t *testing.T) {
	dir := t.TempDir()
	card := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	if _, ok := card.LogoURL(); ok {
		t.Fatal("no logo yet")
	}

	writeFile(t, filepath.Join(card.Dir, "logo.svg"), "<svg/>")
	url, ok := card.LogoURL()
	if !ok || url != "/cards/a/logo.svg" {
		t.Fatalf("unexpected: %q %v", url, ok)
	}
}

func TestCompileJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "card.json")
	writeFile(t, path, `{"title":"Hi","n":2}`)
	c := &Compiler{}
	out, err := c.Compile(Source{Path: path, Kind: KindJSON}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var v map[string]any
	json.Unmarshal(out, &v)
	if v["title"] != "Hi" {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestCompileYAMLNestedMaps(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "card.yaml")
	writeFile(t, path, "title: Hi\nmeta:\n  owner: a\n  tags: [x, y]\n")
	c := &Compiler{}
	out, err := c.Compile(Source{Path: path, Kind: KindYAML}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var v map[string]any
	if err := json.Unmarshal(out, &v); err != nil {
		t.Fatalf("yaml conversion did not produce valid JSON: %v (%s)", err, out)
	}

	m := v["meta"].(map[string]any)
	if m["owner"] != "a" {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestCompileJsonnet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "card.jsonnet")
	writeFile(t, path, `local title = "Hi"; { title: title, n: 1 + 1 }`)
	c := &Compiler{}
	out, err := c.Compile(Source{Path: path, Kind: KindJsonnet}, nil)
	if err != nil {
		t.Fatal(err)
	}

	var v map[string]any
	json.Unmarshal(out, &v)
	if v["title"] != "Hi" || v["n"].(float64) != 2 {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestCompileJsonnetImportsFromSharedDir(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "_shared")
	writeFile(t, filepath.Join(shared, "dossier.libsonnet"), `{ stat(v, l):: { value: v, label: l } }`)
	cardDir := filepath.Join(dir, "a")
	path := filepath.Join(cardDir, "card.jsonnet")
	writeFile(t, path, `local d = import "dossier.libsonnet"; [d.stat("$1", "Revenue")]`)
	c := &Compiler{}
	out, err := c.Compile(Source{Path: path, Kind: KindJsonnet}, []string{cardDir, shared})
	if err != nil {
		t.Fatal(err)
	}

	var v []map[string]any
	json.Unmarshal(out, &v)
	if len(v) != 1 || v[0]["label"] != "Revenue" {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestCompileJsonnetImportsFromImplicitLibDir(t *testing.T) {
	dir := t.TempDir()
	card := Card{ID: "a", Dir: filepath.Join(dir, "a")}
	writeFile(t, filepath.Join(card.Dir, "lib", "helpers.libsonnet"), `{ greet(name):: "hi " + name }`)
	path := filepath.Join(card.Dir, "card.jsonnet")
	writeFile(t, path, `local h = import "helpers.libsonnet"; h.greet("world")`)

	libDir, ok := card.LibDir()
	if !ok {
		t.Fatal("expected lib/ to be found")
	}

	c := &Compiler{}
	out, err := c.Compile(Source{Path: path, Kind: KindJsonnet}, []string{libDir})
	if err != nil {
		t.Fatal(err)
	}

	var v string
	json.Unmarshal(out, &v)
	if v != "hi world" {
		t.Fatalf("unexpected: %s", out)
	}
}

func TestCompileUnsupportedKind(t *testing.T) {
	c := &Compiler{}
	if _, err := c.Compile(Source{Kind: "txt"}, nil); err == nil {
		t.Fatal("expected an error for an unsupported kind")
	}
}
