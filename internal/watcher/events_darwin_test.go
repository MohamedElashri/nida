//go:build darwin

package watcher

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestEventWatcherDetectsFileEdits(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{"content/post.md", "templates/page.html", "nida.toml"} {
		writeWatchFile(t, root, rel, "before")
	}
	changes := startDarwinWatcher(t, root)
	for _, rel := range []string{"content/post.md", "templates/page.html", "nida.toml"} {
		t.Run(rel, func(t *testing.T) {
			// Same-length writes change the file without changing its directory.
			writeWatchFile(t, root, rel, "after!")
			waitForDarwinChange(t, changes, rel)
		})
	}
}

func TestEventWatcherFollowsReplacedFiles(t *testing.T) {
	root := t.TempDir()
	writeWatchFile(t, root, "content/post.md", "before")
	changes := startDarwinWatcher(t, root)
	for i := range 2 {
		// Many editors save by renaming a new file over the original inode.
		writeWatchFile(t, root, "content/.post.tmp", fmt.Sprintf("replacement %d", i))
		if err := os.Rename(filepath.Join(root, "content/.post.tmp"), filepath.Join(root, "content/post.md")); err != nil {
			t.Fatal(err)
		}
		waitForDarwinChange(t, changes, "content/post.md")
		writeWatchFile(t, root, "content/post.md", fmt.Sprintf("edited replacement %d", i))
		waitForDarwinChange(t, changes, "content/post.md")
	}
}

func TestEventWatcherFollowsReplacedDirectories(t *testing.T) {
	root := t.TempDir()
	writeWatchFile(t, root, "content/nested/post.md", "before")
	changes := startDarwinWatcher(t, root)
	if err := os.Rename(filepath.Join(root, "content"), filepath.Join(root, "old")); err != nil {
		t.Fatal(err)
	}
	writeWatchFile(t, root, "content/nested/post.md", "replacement")
	waitForDarwinChange(t, changes, "content/nested/post.md")
	writeWatchFile(t, root, "content/nested/post.md", "edited replacement")
	waitForDarwinChange(t, changes, "content/nested/post.md")
	writeWatchFile(t, root, "content/new/post.md", "new file")
	waitForDarwinChange(t, changes, "content/new/post.md")
	writeWatchFile(t, root, "content/new/post.md", "edited new file")
	waitForDarwinChange(t, changes, "content/new/post.md")
}

func writeWatchFile(t *testing.T, root, rel, body string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func startDarwinWatcher(t *testing.T, root string) <-chan []string {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	changes := make(chan []string, 128)
	done := make(chan error, 1)
	go func() {
		done <- runEventWatcher(ctx, Options{SiteRoot: root, OutputDir: "public", OnChange: func(paths []string) {
			changes <- paths
		}})
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("watcher: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Error("watcher did not stop")
		}
	})
	// Establish readiness through a directory event, without relying on a
	// startup sleep or falling back to the polling watcher.
	for i := range 20 {
		rel := fmt.Sprintf("ready-%d", i)
		writeWatchFile(t, root, rel, "ready")
		select {
		case <-changes:
			return changes
		case err := <-done:
			done <- err
			t.Fatalf("watcher stopped during startup: %v", err)
		case <-time.After(300 * time.Millisecond):
		}
	}
	t.Fatal("watcher did not become ready")
	return nil
}

func waitForDarwinChange(t *testing.T, changes <-chan []string, rel string) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	for {
		select {
		case paths := <-changes:
			if slices.Contains(paths, rel) {
				return
			}
		case <-timer.C:
			t.Fatalf("no change reported for %s", rel)
		}
	}
}
