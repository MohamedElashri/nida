//go:build linux

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

// TestEventWatcherFollowsDirectoryMoves covers directories renamed and
// renamed back, replaced, and moved in with files already inside.
func TestEventWatcherFollowsDirectoryMoves(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	content := filepath.Join(root, "content")
	for _, dir := range []string{filepath.Join(content, "a"), filepath.Join(outside, "in", "sub")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	var seen []string
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- runEventWatcher(ctx, Options{SiteRoot: root, OutputDir: "public", OnChange: func(paths []string) {
			mu.Lock()
			seen = append(seen, paths...)
			mu.Unlock()
		}})
	}()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("watcher: %v", err)
		}
	}()
	time.Sleep(200 * time.Millisecond)

	expect := func(rel string) {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			mu.Lock()
			ok := slices.Contains(seen, rel)
			mu.Unlock()
			if ok {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Fatalf("no event for %s; saw %v", rel, seen)
	}
	write := func(full string) {
		t.Helper()
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	move := func(from, to string) {
		t.Helper()
		if err := os.Rename(from, to); err != nil {
			t.Fatal(err)
		}
	}

	// Renamed and straight back.
	move(filepath.Join(content, "a"), filepath.Join(content, "b"))
	move(filepath.Join(content, "b"), filepath.Join(content, "a"))
	time.Sleep(200 * time.Millisecond)
	write(filepath.Join(content, "a", "one.md"))
	expect("content/a/one.md")

	// Moved away and replaced by a new directory.
	move(filepath.Join(content, "a"), filepath.Join(content, "old"))
	if err := os.Mkdir(filepath.Join(content, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	write(filepath.Join(content, "a", "two.md"))
	expect("content/a/two.md")

	// Moved in with a subdirectory.
	move(filepath.Join(outside, "in"), filepath.Join(content, "in"))
	expect("content/in")
	time.Sleep(200 * time.Millisecond)
	write(filepath.Join(content, "in", "sub", "three.md"))
	expect("content/in/sub/three.md")
}
