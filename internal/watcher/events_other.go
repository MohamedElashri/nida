//go:build !linux && !darwin

package watcher

import (
	"context"
	"errors"
)

func runEventWatcher(ctx context.Context, opts Options) error {
	return errors.New("native filesystem events are unavailable on this platform")
}
