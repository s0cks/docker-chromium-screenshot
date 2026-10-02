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

const SharedDirName = "_shared"

var logoNames = []string{"logo.png", "logo.jpg", "logo.jpeg", "logo.svg", "logo.webp"}

type Card struct {
	ID  string
	Dir string
}

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

func SharedDir(dir string) string { return filepath.Join(dir, SharedDirName) }

type Source struct {
	Path string
	Ext  string // .json | .yaml | .yml | .jsonnet
}

func (c Card) LayoutSource() (Source, bool) {
	for _, ext := range []string{".json", ".yaml", ".yml", ".jsonnet"} {
		p := filepath.Join(c.Dir, "layout"+ext)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return Source{Path: p, Ext: ext}, true
		}
	}

	return Source{}, false
}

func (c Card) HasMarkdown() bool {
	info, err := os.Stat(filepath.Join(c.Dir, "card.md"))
	return err == nil && !info.IsDir()
}

func (c Card) LogoURL() (string, bool) {
	for _, name := range logoNames {
		if info, err := os.Stat(filepath.Join(c.Dir, name)); err == nil && !info.IsDir() {
			return "/cards/" + c.ID + "/" + name, true
		}
	}

	return "", false
}

type Compiler struct{}

func (c *Compiler) Compile(src Source, jpaths []string) (json.RawMessage, error) {
	switch src.Ext {
	case ".json":
		b, err := os.ReadFile(src.Path)
		if err != nil {
			return nil, err
		}

		var v any
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.Marshal(v)

	case ".yaml", ".yml":
		b, err := os.ReadFile(src.Path)
		if err != nil {
			return nil, err
		}

		var raw map[any]any
		if err := yaml.Unmarshal(b, &raw); err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.Marshal(convert(raw))

	case ".jsonnet":
		vm := jsonnet.MakeVM()
		vm.Importer(&jsonnet.FileImporter{JPaths: jpaths})
		out, err := vm.EvaluateFile(src.Path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src.Path, err)
		}

		return json.RawMessage(out), nil

	default:
		return nil, fmt.Errorf("unsupported layout extension %q", src.Ext)
	}
}

func (c *Compiler) EvalJsonnetFile(path string) (string, error) {
	vm := jsonnet.MakeVM()
	vm.Importer(&jsonnet.FileImporter{JPaths: []string{filepath.Dir(path)}})
	return vm.EvaluateFile(path)
}

func InjectLogo(compiled json.RawMessage, logoURL string) (json.RawMessage, error) {
	var doc map[string]any
	if err := json.Unmarshal(compiled, &doc); err != nil {
		return compiled, err
	}

	meta, _ := doc["meta"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		doc["meta"] = meta
	}

	if s, ok := meta["logo"].(string); ok && s != "" {
		return compiled, nil // the card set its own
	}

	meta["logo"] = logoURL
	return json.Marshal(doc)
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
