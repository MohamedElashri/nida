//go:build !linux

package watcher

import "context"

func runEventWatcher(ctx context.Context, opts Options) error {
	return context.Canceled
}
