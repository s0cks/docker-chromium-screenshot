package server

import (
	"encoding/json"
	"log"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/s0cks/docker-chromium-screenshot/service/internal/cards"
	"github.com/s0cks/docker-chromium-screenshot/service/internal/meta"
)

type Config struct {
	Addr     string
	Dist     string // built SPA (astro build output)
	Cards    string // CARDS_DIR: cards/<id>/... plus cards/_shared/...
	Redirect string // path "/" redirects to when set, e.g. "/documents-show/01/"
}

type Server struct {
	cfg      Config
	compiler *cards.Compiler
	mux      *http.ServeMux
	start    time.Time
}

func New(cfg Config) *Server {
	s := &Server{cfg: cfg, compiler: &cards.Compiler{}, mux: http.NewServeMux(), start: time.Now()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)

	s.mux.HandleFunc("/meta", s.handleMeta)
	s.mux.HandleFunc("/meta/", s.handleMeta)

	s.mux.HandleFunc("/layouts", s.handleLayoutList)
	s.mux.HandleFunc("/layout/", s.handleLayout)

	s.mux.Handle("/cards/", http.StripPrefix("/cards/", noDirList(http.Dir(s.cfg.Cards))))

	s.mux.HandleFunc("/", s.handleStatic)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	var id string
	if after, ok := strings.CutPrefix(r.URL.Path, "/meta/"); ok {
		id = strings.TrimSuffix(after, "/")
	}

	payload := meta.Payload{
		Version:   meta.Version,
		Commit:    meta.Commit,
		Branch:    meta.Branch,
		BuildTime: meta.BuildTime,
		GoVersion: runtime.Version(),
	}
	if h := meta.HashDir(s.cfg.Cards); h.Root != "" {
		payload.Hashes = map[string]meta.Hashes{"cards": h}
	}

	extra, err := meta.LoadExtra(cards.SharedDir(s.cfg.Cards), s.compiler.EvalJsonnetFile)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cards/_shared meta: " + err.Error()})
		return
	}

	if id != "" {
		card, ok, err := cards.Find(s.cfg.Cards, id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "no card named " + id})
			return
		}

		cardExtra, err := meta.LoadExtra(card.Dir, s.compiler.EvalJsonnetFile)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": id + " meta: " + err.Error()})
			return
		}

		extra = mergeExtra(extra, cardExtra)
	}
	payload.Extra = extra
	writeJSON(w, http.StatusOK, payload)
}

func mergeExtra(a, b map[string]any) map[string]any {
	if len(a) == 0 {
		return b
	}

	if len(b) == 0 {
		return a
	}

	out := make(map[string]any, len(a)+len(b))
	maps.Copy(out, a)
	maps.Copy(out, b)
	return out
}

func (s *Server) handleLayoutList(w http.ResponseWriter, r *http.Request) {
	list, err := cards.List(s.cfg.Cards)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var ids []string
	for _, c := range list {
		if _, ok := c.LayoutSource(); ok {
			ids = append(ids, c.ID)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"layouts": ids,
	})
}

func (s *Server) handleLayout(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/layout/"), "/")
	card, ok, err := cards.Find(s.cfg.Cards, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no card named " + id})
		return
	}

	src, ok := card.LayoutSource()
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": id + " has no layout.json/yaml/jsonnet"})
		return
	}

	out, err := s.compiler.Compile(src, []string{card.Dir, cards.SharedDir(s.cfg.Cards)})
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	if logoURL, ok := card.LogoURL(); ok {
		out, err = cards.InjectLogo(out, logoURL)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(out)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" && s.cfg.Redirect != "" {
		http.Redirect(w, r, s.cfg.Redirect, http.StatusFound)
		return
	}

	fs := http.Dir(s.cfg.Dist)
	upath := r.URL.Path
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}

	name := filepath.Clean(strings.TrimPrefix(upath, "/"))
	full := filepath.Join(s.cfg.Dist, name)
	if info, err := os.Stat(full); err == nil {
		if info.IsDir() {
			if _, err := os.Stat(filepath.Join(full, "index.html")); err == nil {
				http.ServeFile(w, r, filepath.Join(full, "index.html"))
				return
			}
		} else {
			http.FileServer(fs).ServeHTTP(w, r)
			return
		}
	}

	if filepath.Ext(name) == "" {
		if _, err := os.Stat(filepath.Join(s.cfg.Dist, "index.html")); err == nil {
			http.ServeFile(w, r, filepath.Join(s.cfg.Dist, "index.html"))
			return
		}
	}

	http.NotFound(w, r)
}

func noDirList(root http.Dir) http.Handler {
	fs := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := root.Open(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		if info, err := f.Stat(); err == nil && info.IsDir() {
			http.NotFound(w, r)
			return
		}

		fs.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}
