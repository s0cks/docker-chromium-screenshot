package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v2"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	Branch    = "unknown"
	BuildTime = "unknown"
)

type Hashes struct {
	Root  string            `json:"root,omitempty"`
	Files map[string]string `json:"files,omitempty"`
}

func HashDir(dir string) Hashes {
	files := map[string]string{}
	if dir == "" {
		return Hashes{}
	}

	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || len(d.Name()) == 0 || d.Name()[0] == '.' {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = path
		}

		sum := sha256.Sum256(b)
		files[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})

	if len(files) == 0 {
		return Hashes{}
	}

	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}

	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte(" "))
		h.Write([]byte(files[p]))
		h.Write([]byte("\n"))
	}

	return Hashes{Root: hex.EncodeToString(h.Sum(nil)), Files: files}
}

type Payload struct {
	Version   string            `json:"version"`
	Commit    string            `json:"commit"`
	Branch    string            `json:"branch"`
	BuildTime string            `json:"buildTime"`
	GoVersion string            `json:"goVersion"`
	Hashes    map[string]Hashes `json:"hashes,omitempty"`
	Extra     map[string]any    `json:"extra,omitempty"`
}

func LoadExtra(dir string, eval func(path string) (string, error)) (map[string]any, error) {
	if dir == "" {
		return nil, nil
	}

	candidates := []struct {
		name   string
		decode func([]byte) (map[string]any, error)
	}{
		{"meta.json", decodeJSON},
		{"meta.yaml", decodeYAML},
		{"meta.yml", decodeYAML},
		{"meta.jsonnet", nil}, // handled below via eval
	}

	for _, c := range candidates {
		path := filepath.Join(dir, c.name)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}

		if c.decode == nil {
			if eval == nil {
				continue
			}

			out, err := eval(path)
			if err != nil {
				return nil, err
			}

			return decodeJSON([]byte(out))
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		return c.decode(b)
	}

	return nil, nil
}

func decodeJSON(b []byte) (map[string]any, error) {
	var m map[string]any
	if len(b) == 0 {
		return nil, nil
	}

	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func decodeYAML(b []byte) (map[string]any, error) {
	var raw map[any]any
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
	}

	out, _ := normalizeYAML(raw).(map[string]any)
	return out, nil
}

func normalizeYAML(v any) any {
	switch v := v.(type) {
	case map[any]any:
		m := make(map[string]any, len(v))
		for k, val := range v {
			m[toString(k)] = normalizeYAML(val)
		}

		return m

	case []any:
		out := make([]any, len(v))
		for i, val := range v {
			out[i] = normalizeYAML(val)
		}

		return out

	default:
		return v
	}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	b, _ := json.Marshal(v)
	return string(b)
}
