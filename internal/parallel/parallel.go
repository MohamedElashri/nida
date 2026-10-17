// Package parallel runs independent work across CPU cores.
package parallel

import (
	"runtime"
	"sync"
)

// Map applies fn to every item using up to GOMAXPROCS goroutines and
// returns results in input order. If any call fails, it returns the error
// from the lowest index, so failures are reported the same way as a
// sequential loop would report them.
func Map[T, R any](items []T, fn func(T) (R, error)) ([]R, error) {
	results := make([]R, len(items))
	if len(items) == 0 {
		return results, nil
	}
	errs := make([]error, len(items))
	workers := min(runtime.GOMAXPROCS(0), len(items))

	var next int
	var mu sync.Mutex
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				mu.Lock()
				i := next
				next++
				mu.Unlock()
				if i >= len(items) {
					return
				}
				results[i], errs[i] = fn(items[i])
			}
		}()
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return results, nil
}
