//go:build linux

package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func runEventWatcher(ctx context.Context, opts Options) error {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		return err
	}
	defer func() { _ = syscall.Close(fd) }()

	absSiteRoot, err := filepath.Abs(opts.SiteRoot)
	if err != nil {
		return fmt.Errorf("resolve site root %q: %w", opts.SiteRoot, err)
	}
	outputRoot, err := resolveOutputRoot(absSiteRoot, opts.OutputDir)
	if err != nil {
		return err
	}

	watches := map[int]string{}
	dirs := map[string]int{}
	addWatch := func(dir string) error {
		dir = filepath.Clean(dir)
		if shouldSkipPath(dir, outputRoot) {
			return nil
		}
		if _, ok := dirs[dir]; ok {
			return nil
		}
		mask := uint32(syscall.IN_CREATE | syscall.IN_MODIFY | syscall.IN_DELETE | syscall.IN_DELETE_SELF | syscall.IN_MOVE_SELF | syscall.IN_MOVED_FROM | syscall.IN_MOVED_TO | syscall.IN_ATTRIB | syscall.IN_CLOSE_WRITE)
		wd, err := syscall.InotifyAddWatch(fd, dir, mask)
		if err != nil {
			return err
		}
		watches[wd] = dir
		dirs[dir] = wd
		return nil
	}

	if err := filepath.WalkDir(absSiteRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		if shouldSkipPath(path, outputRoot) {
			return fs.SkipDir
		}
		return addWatch(path)
	}); err != nil {
		return err
	}

	_ = syscall.SetNonblock(fd, true)
	events := make(chan string, 128)
	errorsCh := make(chan error, 1)

	// The reader must stop before fd is closed: a new watcher may get the
	// same descriptor number, and a reader left running would steal its
	// events.
	readerCtx, stopReader := context.WithCancel(ctx)
	readerDone := make(chan struct{})
	defer func() {
		stopReader()
		<-readerDone
	}()

	// send reports a changed path relative to the site root, and returns
	// false once the watcher is stopping.
	send := func(fullPath string) bool {
		rel, err := filepath.Rel(absSiteRoot, fullPath)
		if err != nil {
			return true
		}
		rel = filepath.ToSlash(rel)
		if rel == "." || strings.HasPrefix(rel, "../") {
			return true
		}
		select {
		case events <- rel:
			return true
		case <-readerCtx.Done():
			return false
		}
	}

	go func() {
		defer close(readerDone)
		buf := make([]byte, 64*1024)
		for readerCtx.Err() == nil {
			n, readErr := syscall.Read(fd, buf)
			if readErr != nil {
				if readErr == syscall.EAGAIN {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				select {
				case errorsCh <- readErr:
				default:
				}
				return
			}
			if n <= 0 {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			offset := 0
			for offset < n {
				raw := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[offset]))
				offset += syscall.SizeofInotifyEvent
				nameBytes := buf[offset : offset+int(raw.Len)]
				offset += int(raw.Len)
				name := strings.TrimRight(string(nameBytes), "\x00")

				if raw.Mask&syscall.IN_IGNORED != 0 {
					// The directory was deleted or unwatched; forget it so a
					// directory created at the same path is watched again.
					if dir, ok := watches[int(raw.Wd)]; ok {
						delete(watches, int(raw.Wd))
						if dirs[dir] == int(raw.Wd) {
							delete(dirs, dir)
						}
					}
					continue
				}

				dir, ok := watches[int(raw.Wd)]
				if !ok {
					continue
				}
				fullPath := dir
				if name != "" {
					fullPath = filepath.Join(dir, name)
				}

				if raw.Mask&syscall.IN_ISDIR != 0 {
					switch {
					case raw.Mask&(syscall.IN_CREATE|syscall.IN_MOVED_TO) != 0:
						// A directory moved in arrives with its files and
						// subdirectories, which send no events of their own,
						// and files created in a new directory before its
						// watch is added send none either. Each directory is
						// watched before it is listed, so every file is
						// either reported here or sends its own event.
						stopped := false
						_ = filepath.WalkDir(fullPath, func(path string, d fs.DirEntry, err error) error {
							if err != nil {
								return nil
							}
							if shouldSkipPath(path, outputRoot) {
								if d.IsDir() {
									return fs.SkipDir
								}
								return nil
							}
							if d.IsDir() {
								return addWatch(path)
							}
							if !send(path) {
								stopped = true
								return fs.SkipAll
							}
							return nil
						})
						if stopped {
							return
						}
					case raw.Mask&syscall.IN_MOVED_FROM != 0:
						// A directory moved out stays watched where it went.
						// Forget it now, not when IN_IGNORED arrives, so a
						// directory moved or created at the same path in
						// the meantime is watched.
						for path, wd := range dirs {
							if path == fullPath || strings.HasPrefix(path, fullPath+string(filepath.Separator)) {
								_, _ = syscall.InotifyRmWatch(fd, uint32(wd))
								delete(dirs, path)
								delete(watches, wd)
							}
						}
					case raw.Mask&syscall.IN_DELETE == 0:
						continue
					}
				}

				if !send(fullPath) {
					return
				}
			}
		}
	}()

	pending := map[string]struct{}{}
	var debounce <-chan time.Time

	flush := func() {
		if len(pending) == 0 {
			return
		}
		changed := make([]string, 0, len(pending))
		for path := range pending {
			changed = append(changed, path)
		}
		sort.Strings(changed)
		pending = map[string]struct{}{}
		opts.OnChange(changed)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errorsCh:
			return err
		case path := <-events:
			pending[path] = struct{}{}
			debounce = time.After(150 * time.Millisecond)
		case <-debounce:
			flush()
			debounce = nil
		}
	}
}
