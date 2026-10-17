package content

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestApplyGitDates(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	run := func(env []string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), env...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "content", "posts"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, "content", "posts", name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ident := []string{"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com"}
	run(nil, "init", "-q")
	run(nil, "add", ".")
	run(append(ident, "GIT_COMMITTER_DATE=2026-01-02T03:04:05Z"), "commit", "-q", "-m", "one")
	if err := os.WriteFile(filepath.Join(root, "content", "posts", "a.md"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(nil, "add", ".")
	run(append(ident, "GIT_COMMITTER_DATE=2026-03-04T05:06:07Z"), "commit", "-q", "-m", "two")

	keep := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	pages := []Page{
		{RelativePath: "posts/a.md"},
		{RelativePath: "posts/b.md"},
		{RelativePath: "posts/b.md", Updated: keep},
	}
	ApplyGitDates(filepath.Join(root, "content"), pages)

	if got := pages[0].Updated.UTC(); !got.Equal(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)) {
		t.Fatalf("a.md updated = %v", got)
	}
	if got := pages[1].Updated.UTC(); !got.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("b.md updated = %v", got)
	}
	if !pages[2].Updated.Equal(keep) {
		t.Fatalf("front matter updated must win, got %v", pages[2].Updated)
	}
}
