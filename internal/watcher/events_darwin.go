//go:build darwin

package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"syscall"
	"time"
)

func runEventWatcher(ctx context.Context, opts Options) error {
	absSiteRoot, err := filepath.Abs(opts.SiteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", opts.SiteRoot, err)
	}
	outputRoot := filepath.Join(absSiteRoot, opts.OutputDir)

	kq, err := syscall.Kqueue()
	if err != nil {
		return err
	}
	defer syscall.Close(kq)

	type watchEntry struct {
		fd   int
		path string
	}

	watches := map[string]watchEntry{}
	previous, err := snapshot(opts.SiteRoot, opts.OutputDir)
	if err != nil {
		return err
	}

	syncWatches := func() error {
		currentDirs := map[string]struct{}{}
		err := filepath.WalkDir(absSiteRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if !d.IsDir() {
				return nil
			}
			path = filepath.Clean(path)
			if shouldSkipPath(path, outputRoot) {
				return fs.SkipDir
			}
			currentDirs[path] = struct{}{}
			if _, ok := watches[path]; ok {
				return nil
			}

			fd, err := syscall.Open(path, syscall.O_RDONLY, 0)
			if err != nil {
				return err
			}

			event := syscall.Kevent_t{
				Ident:  uint64(fd),
				Filter: syscall.EVFILT_VNODE,
				Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_CLEAR,
				Fflags: syscall.NOTE_WRITE | syscall.NOTE_DELETE | syscall.NOTE_EXTEND | syscall.NOTE_ATTRIB | syscall.NOTE_LINK | syscall.NOTE_RENAME | syscall.NOTE_REVOKE,
			}
			if _, err := syscall.Kevent(kq, []syscall.Kevent_t{event}, nil, nil); err != nil {
				_ = syscall.Close(fd)
				return err
			}

			watches[path] = watchEntry{fd: fd, path: path}
			return nil
		})
		if err != nil {
			return err
		}

		for path, entry := range watches {
			if _, ok := currentDirs[path]; ok {
				continue
			}
			_, _ = syscall.Kevent(kq, []syscall.Kevent_t{{
				Ident:  uint64(entry.fd),
				Filter: syscall.EVFILT_VNODE,
				Flags:  syscall.EV_DELETE,
			}}, nil, nil)
			_ = syscall.Close(entry.fd)
			delete(watches, path)
		}

		return nil
	}

	if err := syncWatches(); err != nil {
		return err
	}
	defer func() {
		for _, entry := range watches {
			_ = syscall.Close(entry.fd)
		}
	}()

	events := make([]syscall.Kevent_t, 64)
	timeout := syscall.NsecToTimespec((250 * time.Millisecond).Nanoseconds())
	pending := false
	var debounce <-chan time.Time

	flush := func() error {
		current, err := snapshot(opts.SiteRoot, opts.OutputDir)
		if err != nil {
			return err
		}
		changed := diff(previous, current)
		if len(changed) > 0 {
			opts.OnChange(changed)
			previous = current
		}
		return syncWatches()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-debounce:
			if pending {
				if err := flush(); err != nil {
					return err
				}
				pending = false
			}
			debounce = nil
		default:
		}

		n, err := syscall.Kevent(kq, nil, events, &timeout)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return err
		}
		if n == 0 {
			continue
		}

		changedDir := false
		for _, event := range events[:n] {
			if event.Flags&syscall.EV_ERROR != 0 {
				return fmt.Errorf("kqueue watcher error on fd %d: %d", event.Ident, event.Data)
			}
			changedDir = true
		}
		if changedDir {
			pending = true
			debounce = time.After(150 * time.Millisecond)
		}
	}
}
