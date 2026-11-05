//go:build darwin

package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

func runEventWatcher(ctx context.Context, opts Options) error {
	absSiteRoot, err := filepath.Abs(opts.SiteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", opts.SiteRoot, err)
	}
	outputRoot, err := resolveOutputRoot(absSiteRoot, opts.OutputDir)
	if err != nil {
		return err
	}

	kq, err := syscall.Kqueue()
	if err != nil {
		return err
	}
	defer syscall.Close(kq)

	type watchEntry struct {
		fd   int
		info fs.FileInfo
	}

	watches := map[string]watchEntry{}
	defer func() {
		for _, entry := range watches {
			_ = syscall.Close(entry.fd)
		}
	}()
	removeWatch := func(path string, entry watchEntry) {
		_, _ = syscall.Kevent(kq, []syscall.Kevent_t{{
			Ident:  uint64(entry.fd),
			Filter: syscall.EVFILT_VNODE,
			Flags:  syscall.EV_DELETE,
		}}, nil, nil)
		_ = syscall.Close(entry.fd)
		delete(watches, path)
	}

	syncWatches := func() error {
		currentPaths := map[string]struct{}{}
		err := filepath.WalkDir(absSiteRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			path = filepath.Clean(path)
			if shouldSkipPath(path, outputRoot) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			// Directory vnode events report entry changes, but not writes
			// to existing files. Watch regular files individually as well.
			// Do not follow symlinks or open special files.
			if !d.IsDir() && !d.Type().IsRegular() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			currentPaths[path] = struct{}{}
			if entry, ok := watches[path]; ok {
				if os.SameFile(entry.info, info) {
					return nil
				}
				// An atomic save or directory replacement leaves the old
				// descriptor watching an inode no longer at this path.
				removeWatch(path, entry)
			}

			fd, err := syscall.Open(path, syscall.O_EVTONLY|syscall.O_NOFOLLOW, 0)
			if err != nil {
				if os.IsNotExist(err) || err == syscall.ELOOP {
					return nil
				}
				return err
			}
			// If the path changed between stat and open, its parent event
			// will trigger another pass to register the replacement.
			var stat syscall.Stat_t
			if err := syscall.Fstat(fd, &stat); err != nil {
				_ = syscall.Close(fd)
				return err
			}
			if original, ok := info.Sys().(*syscall.Stat_t); !ok || original.Dev != stat.Dev || original.Ino != stat.Ino {
				_ = syscall.Close(fd)
				return nil
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

			watches[path] = watchEntry{fd: fd, info: info}
			return nil
		})
		if err != nil {
			return err
		}

		for path, entry := range watches {
			if _, ok := currentPaths[path]; ok {
				continue
			}
			removeWatch(path, entry)
		}

		return nil
	}

	if err := syncWatches(); err != nil {
		return err
	}
	previous, err := snapshot(opts.SiteRoot, opts.OutputDir)
	if err != nil {
		return err
	}

	events := make([]syscall.Kevent_t, 64)
	timeout := syscall.NsecToTimespec((250 * time.Millisecond).Nanoseconds())
	pending := false
	var debounce <-chan time.Time

	flush := func() error {
		if err := syncWatches(); err != nil {
			return err
		}
		current, err := snapshot(opts.SiteRoot, opts.OutputDir)
		if err != nil {
			return err
		}
		changed := diff(previous, current)
		if len(changed) > 0 {
			opts.OnChange(changed)
			previous = current
		}
		return nil
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

		for _, event := range events[:n] {
			if event.Flags&syscall.EV_ERROR != 0 {
				return fmt.Errorf("kqueue watcher error on fd %d: %d", event.Ident, event.Data)
			}
		}
		pending = true
		debounce = time.After(150 * time.Millisecond)
	}
}
