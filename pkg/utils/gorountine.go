package utils

import (
	"context"
	"sync"
)

// RunParallel runs multiple functions concurrently.
// It returns the first error encountered, or nil if all functions succeed.
func RunParallel(ctx context.Context, limit int, funcs ...func(ctx context.Context) error) error {
	if limit <= 0 {
		limit = len(funcs)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg    sync.WaitGroup
		sem   = make(chan struct{}, limit) // concurrency limiter
		errCh = make(chan error, len(funcs))
	)

	// Launch all functions
	wg.Add(len(funcs))
	for _, f := range funcs {
		go func(fn func(ctx context.Context) error) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return // Stop if context is canceled
			case sem <- struct{}{}:
				defer func() { <-sem }() // Release semaphore slot when done
			}

			if err := fn(ctx); err != nil {
				select {
				case errCh <- err:
					cancel() // cancel others
				default:
				}
			}
		}(f)
	}

	// Wait for all goroutines to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(errCh)
	}()
	// wg.Wait()
	// close(errCh)

	// Monitor both the context and error channel
	select {
	case <-ctx.Done():
		// If the context was canceled due to an error, drain the channel to get it
		for err := range errCh {
			if err != nil {
				return err
			}
		}
		return ctx.Err()
	case err, ok := <-errCh:
		if ok && err != nil {
			return err
		}
		return nil
	}
}
