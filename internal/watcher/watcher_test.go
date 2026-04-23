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

func TestSnapshotSkipsGitDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatalf("mkdir content: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "objects", "blob"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "post.md"), []byte("tracked"), 0o644); err != nil {
		t.Fatalf("write content: %v", err)
	}

	got, err := snapshot(dir, "public")
	if err != nil {
		t.Fatalf("snapshot returned error: %v", err)
	}
	if _, ok := got[".git/objects/blob"]; ok {
		t.Fatalf("expected .git file to be skipped, got %+v", got)
	}
	if _, ok := got["content/post.md"]; !ok {
		t.Fatalf("expected content file in snapshot, got %+v", got)
	}
}

func TestShouldSkipPathSkipsVCSSegments(t *testing.T) {
	if !shouldSkipPath(filepath.Join("site", ".git"), "") {
		t.Fatal("expected .git directory to be skipped")
	}
	if !shouldSkipPath(filepath.Join("site", ".git", "objects", "pack"), "") {
		t.Fatal("expected path inside .git to be skipped")
	}
	if !shouldSkipPath(filepath.Join("site", "content", ".svn", "entries"), "") {
		t.Fatal("expected path inside .svn to be skipped")
	}
	if shouldSkipPath(filepath.Join("site", "content", "post.md"), "") {
		t.Fatal("did not expect normal content path to be skipped")
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

func TestShouldSkipPathSkipsVCSDirectoriesRecursively(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "site")
	output := filepath.Join(root, "public")

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "output root", path: output, want: true},
		{name: "output child", path: filepath.Join(output, "index.html"), want: true},
		{name: "git root", path: filepath.Join(root, ".git"), want: true},
		{name: "git nested", path: filepath.Join(root, ".git", "objects", "ab"), want: true},
		{name: "regular dotfile", path: filepath.Join(root, ".gitignore"), want: false},
		{name: "content file", path: filepath.Join(root, "content", "post.md"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldSkipPath(tt.path, output); got != tt.want {
				t.Fatalf("shouldSkipPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestSnapshotSkipsGitDirectoryButKeepsDotfiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "content"), 0o755); err != nil {
		t.Fatalf("mkdir content: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git", "objects"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "content", "post.md"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write content: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "objects", "pack"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("write git object: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("public"), 0o644); err != nil {
		t.Fatalf("write dotfile: %v", err)
	}

	got, err := snapshot(dir, "public")
	if err != nil {
		t.Fatalf("snapshot returned error: %v", err)
	}
	if _, ok := got[".git/objects/pack"]; ok {
		t.Fatalf("expected .git file to be skipped, got %+v", got)
	}
	if _, ok := got[".gitignore"]; !ok {
		t.Fatalf("expected .gitignore to be included, got %+v", got)
	}
	if _, ok := got["content/post.md"]; !ok {
		t.Fatalf("expected content file in snapshot, got %+v", got)
	}
}
