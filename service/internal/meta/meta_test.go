package meta

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashDirEmptyAndMissing(t *testing.T) {
	if h := HashDir(""); h.Root != "" {
		t.Fatalf("empty dir arg: want zero value, got %+v", h)
	}
	if h := HashDir(filepath.Join(t.TempDir(), "nope")); h.Root != "" {
		t.Fatalf("missing dir: want zero value, got %+v", h)
	}
}

func TestHashDirDeterministic(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("alpha"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("beta"), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden"), []byte("skip me"), 0o644)

	h1 := HashDir(dir)
	h2 := HashDir(dir)
	if h1.Root == "" {
		t.Fatal("expected a non-empty root hash")
	}
	if h1.Root != h2.Root {
		t.Fatalf("hashing twice should be stable: %s != %s", h1.Root, h2.Root)
	}
	if len(h1.Files) != 2 {
		t.Fatalf("dotfiles should be skipped: got %d files, want 2", len(h1.Files))
	}
	if _, ok := h1.Files[".hidden"]; ok {
		t.Fatal(".hidden should not be hashed")
	}

	os.WriteFile(filepath.Join(dir, "a.md"), []byte("ALPHA"), 0o644)
	if h3 := HashDir(dir); h3.Root == h1.Root {
		t.Fatal("changing a file's content should change the root hash")
	}
}

func TestLoadExtraPrefersJSONThenYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "meta.yaml"), []byte("owner: yaml-owner\n"), 0o644)
	extra, err := LoadExtra(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if extra["owner"] != "yaml-owner" {
		t.Fatalf("want owner from yaml, got %v", extra)
	}

	os.WriteFile(filepath.Join(dir, "meta.json"), []byte(`{"owner":"json-owner","nested":{"k":1}}`), 0o644)
	extra, err = LoadExtra(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if extra["owner"] != "json-owner" {
		t.Fatalf("meta.json should take priority over meta.yaml, got %v", extra)
	}
}

func TestLoadExtraJsonnet(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "meta.jsonnet"), []byte(`{ owner: "jsonnet-owner", n: 1 + 1 }`), 0o644)
	extra, err := LoadExtra(dir, func(path string) (string, error) {
		return `{"owner":"jsonnet-owner","n":2}`, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if extra["owner"] != "jsonnet-owner" || extra["n"].(float64) != 2 {
		t.Fatalf("unexpected extra: %v", extra)
	}
}

func TestLoadExtraNoFiles(t *testing.T) {
	extra, err := LoadExtra(t.TempDir(), nil)
	if err != nil || extra != nil {
		t.Fatalf("want (nil, nil) when no meta file exists, got (%v, %v)", extra, err)
	}
}
