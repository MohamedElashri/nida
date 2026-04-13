package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSnapshotSkipsOutputDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatalf("mkdir content: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "public"), 0o755); err != nil {
		t.Fatalf("mkdir public: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "post.md"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "public", "index.html"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write public: %v", err)
	}

	got, err := snapshot(dir, "public")
	if err != nil {
		t.Fatalf("snapshot returned error: %v", err)
	}
	if _, ok := got["public/index.html"]; ok {
		t.Fatalf("expected output file to be skipped, got %+v", got)
	}
	if _, ok := got["content/post.md"]; !ok {
		t.Fatalf("expected content file in snapshot, got %+v", got)
	}
}

func TestDiffDetectsChangesAndDeletions(t *testing.T) {
	now := time.Now().UTC()
	previous := map[string]fileState{
		"a.txt": {size: 1, modTime: now},
		"b.txt": {size: 2, modTime: now},
	}
	current := map[string]fileState{
		"a.txt": {size: 2, modTime: now},
		"c.txt": {size: 3, modTime: now},
	}

	got := diff(previous, current)
	want := []string{"a.txt", "b.txt", "c.txt"}
	if len(got) != len(want) {
		t.Fatalf("unexpected diff length: got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected diff: got %v want %v", got, want)
		}
	}
}

func TestRunRequiresOnChange(t *testing.T) {
	err := Run(t.Context(), Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}
