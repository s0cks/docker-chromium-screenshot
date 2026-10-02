package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dist := t.TempDir()
	writeFile(t, filepath.Join(dist, "index.html"), "<html>home</html>")
	writeFile(t, filepath.Join(dist, "showcase", "01", "index.html"), "<html>slide 1</html>")

	cardsDir := t.TempDir()
	writeFile(t, filepath.Join(cardsDir, "demo", "layout.json"), `{"meta":{"series":"Demo"},"slides":[{"title":"One"}]}`)
	writeFile(t, filepath.Join(cardsDir, "demo", "logo.svg"), "<svg/>")
	writeFile(t, filepath.Join(cardsDir, "demo", "meta.json"), `{"owner":"card-owner"}`)
	writeFile(t, filepath.Join(cardsDir, "documents-show", "card.md"), "# Markdown card, not a layout")
	writeFile(t, filepath.Join(cardsDir, "_shared", "theme.css"), ":root{--accent:red}")
	writeFile(t, filepath.Join(cardsDir, "_shared", "meta.json"), `{"owner":"shared-owner","org":"acme"}`)

	return New(Config{Dist: dist, Cards: cardsDir})
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestMetaServesBuildInfoAndSharedExtra(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/meta", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body["version"] == nil {
		t.Fatalf("missing version: %s", rec.Body)
	}
	hashes := body["hashes"].(map[string]interface{})
	if _, ok := hashes["cards"]; !ok {
		t.Fatalf("expected a cards hash: %s", rec.Body)
	}
	extra := body["extra"].(map[string]interface{})
	if extra["owner"] != "shared-owner" || extra["org"] != "acme" {
		t.Fatalf("expected shared meta, got %v", extra)
	}
}

func TestMetaForCardMergesSharedAndCardOwnMeta(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/meta/demo", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)
	extra := body["extra"].(map[string]interface{})
	if extra["owner"] != "card-owner" {
		t.Fatalf("card's own meta should win over shared: %v", extra)
	}
	if extra["org"] != "acme" {
		t.Fatalf("shared-only keys should still be present: %v", extra)
	}
}

func TestMetaForUnknownCardIs404(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/meta/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLayoutServesCompiledJSONWithInjectedLogo(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/layout/demo", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	var body map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &body)
	meta := body["meta"].(map[string]interface{})
	if meta["series"] != "Demo" {
		t.Fatalf("unexpected: %s", rec.Body)
	}
	if meta["logo"] != "/cards/demo/logo.svg" {
		t.Fatalf("expected the card's own logo to be injected, got %v", meta["logo"])
	}
}

func TestLayoutMissingIs404(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/layout/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestLayoutOnMarkdownCardIs404(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/layout/documents-show", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("a card.md card has no layout.*, want 404, got %d", rec.Code)
	}
}

func TestLayoutList(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/layouts", nil))
	var body map[string][]string
	json.Unmarshal(rec.Body.Bytes(), &body)
	if len(body["layouts"]) != 1 || body["layouts"][0] != "demo" {
		t.Fatalf("unexpected: %s (documents-show has no layout.* and should be excluded)", rec.Body)
	}
}

func TestCardsServesFileNotDirectory(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cards/_shared/theme.css", nil))
	if rec.Code != http.StatusOK || rec.Body.String() == "" {
		t.Fatalf("status = %d, body = %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cards/demo/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("directory listing should 404, got %d", rec.Code)
	}
}

func TestCardsServesLogo(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cards/demo/logo.svg", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStaticServesIndexAndNestedSlide(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/showcase/01/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestStaticFallsBackToIndexForExtensionlessPath(t *testing.T) {
	s := newTestServer(t)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/live/showcase/01", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestRootRedirect(t *testing.T) {
	dist := t.TempDir()
	writeFile(t, filepath.Join(dist, "index.html"), "home")
	s := New(Config{Dist: dist, Redirect: "/showcase/01/"})
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/showcase/01/" {
		t.Fatalf("status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
}
