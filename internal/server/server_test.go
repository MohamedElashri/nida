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
	if body := rec.Body.String(); !strings.Contains(body, `/_nida/livereload`) {
		t.Fatalf("expected livereload script in response, got %q", body)
	}
}
