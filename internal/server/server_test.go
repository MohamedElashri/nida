package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileHandlerServesOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	handler, err := FileHandler(dir)
	if err != nil {
		t.Fatalf("FileHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Body.String() != "hello" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestFileHandlerServesCustom404PageForMissingRoutes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "404.html"), []byte("custom not found"), 0o644); err != nil {
		t.Fatalf("write 404: %v", err)
	}

	handler, err := FileHandler(dir)
	if err != nil {
		t.Fatalf("FileHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/missing/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 status, got %d", rec.Code)
	}
	if rec.Body.String() != "custom not found" {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestFileHandlerRejectsSymlinkedFiles(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(secret, []byte("secret"), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(dir, "secret.txt")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	handler, err := FileHandler(dir)
	if err != nil {
		t.Fatalf("FileHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/secret.txt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code == http.StatusOK || strings.Contains(rec.Body.String(), "secret") {
		t.Fatalf("expected symlinked file not to be served, status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestFileHandlerInjectsLiveReloadScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><html><body><main>hello</main></body></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	handler, err := fileHandler(dir, true, newReloadBroker())
	if err != nil {
		t.Fatalf("fileHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, `<script src="/_nida/livereload.js"></script>`) {
		t.Fatalf("expected livereload script in response, got %q", body)
	}
}

func TestFileHandlerServesExternalLiveReloadScript(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><html><body><main>hello</main></body></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	handler, err := fileHandler(dir, true, newReloadBroker())
	if err != nil {
		t.Fatalf("fileHandler returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/_nida/livereload.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", rec.Code)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.Contains(contentType, "text/javascript") {
		t.Fatalf("expected javascript content type, got %q", contentType)
	}
	if body := rec.Body.String(); !strings.Contains(body, `new EventSource("/_nida/livereload")`) {
		t.Fatalf("expected livereload client script, got %q", body)
	}
}

func FileHandler(outputDir string) (http.Handler, error) {
	return fileHandler(outputDir, false, nil)
}

func fileHandler(outputDir string, injectLiveReload bool, reloader *reloadBroker) (http.Handler, error) {
	return fileHandlerWithBasePath(outputDir, "", injectLiveReload, reloader)
}

func TestWithBasePathStripsDecodedBasePath(t *testing.T) {
	seen := ""
	handler := withBasePath(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.URL.Path }), "/my%20blog")
	for target, want := range map[string]string{
		"/my%20blog":              "/",
		"/my%20blog/":             "/",
		"/my%20blog/posts/a%20b/": "/posts/a b/",
		"/my%20blogger/":          "/my blogger/",
		"/other/":                 "/other/",
	} {
		seen = ""
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
		if seen != want {
			t.Errorf("%s: handler saw %q, want %q", target, seen, want)
		}
	}
}

func TestFileHandlerDoesNotListDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "posts", "gone"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "posts", "gone", "note.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	handler, err := FileHandler(dir)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/gone/", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for a directory without index.html, got %d: %s", rec.Code, rec.Body)
	}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/posts/gone/note.txt", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected the file itself to be served, got %d", rec.Code)
	}
}
