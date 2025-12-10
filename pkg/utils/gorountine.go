package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
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
			defer func() {
				if r := recover(); r != nil {
					select {
					case errCh <- fmt.Errorf("panic: %v", r):
						cancel() // cancel others
					default:
					}
				}
			}()

			defer wg.Done()
			select {
			case <-ctx.Done():
				return // Stop if context is cancelled
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
		// If the context was cancelled due to an error, drain the channel to get it
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

// SafeFunc runs a function and recovers from any panic.
func SafeFunc(ctx context.Context, fn func(ctx context.Context) error) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic
			zap.L().Error("Recovered from panic in SafeGo", zap.Any("panic", r))
		}
	}()

	fn(ctx)
}

type RetryOptions struct {
	MaxAttempts       int
	BaseBackoff       time.Duration
	BackoffMultiplier float64
	AttemptTimeout    time.Duration
}

// DefaultRetryOptions provides sensible defaults for retry behavior.
//   - MaxAttempts: 3
//   - BaseBackoff: 1 second
//   - BackoffMultiplier: 1.5
//   - AttemptTimeout: 20 seconds
var DefaultRetryOptions = RetryOptions{
	MaxAttempts:       3,
	BaseBackoff:       1 * time.Second,
	BackoffMultiplier: 1.5,
	AttemptTimeout:    20 * time.Second,
}

// MinimalRetryOptions provides minimal retry settings.
//   - MaxAttempts: 1
//   - BaseBackoff: 1 second
//   - BackoffMultiplier: 1.5
//   - AttemptTimeout: 20 seconds
var MinimalRetryOptions = RetryOptions{
	MaxAttempts:       1,
	BaseBackoff:       1 * time.Second,
	BackoffMultiplier: 1.5,
	AttemptTimeout:    20 * time.Second,
}

func RunParallelWithRetry(
	ctx context.Context,
	limit int,
	opts RetryOptions,
	funcs ...func(ctx context.Context) error,
) error {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = time.Second
	}
	if opts.BackoffMultiplier <= 0 {
		opts.BackoffMultiplier = 1.5
	}
	if opts.AttemptTimeout <= 0 {
		opts.AttemptTimeout = 20 * time.Second
	}

	if limit <= 0 {
		limit = len(funcs)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg    sync.WaitGroup
		sem   = make(chan struct{}, limit)
		errCh = make(chan error, 1)
	)

	wg.Add(len(funcs))

	for jobID, fn := range funcs {

		go func(job int, fn func(ctx context.Context) error) {
			defer wg.Done()

			// Panic recovery per job
			defer func() {
				if r := recover(); r != nil {
					select {
					case errCh <- fmt.Errorf("panic in job %d: %v", job, r):
					default:
					}
					cancel()
				}
			}()

			// Acquire concurrency slot
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			var lastErr error

			for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {

				// Per-attempt timeout
				attemptCtx, attemptCancel := context.WithTimeout(ctx, opts.AttemptTimeout)

				lastErr = fn(attemptCtx)
				attemptCancel()

				if lastErr == nil {
					return // success
				}

				// Failed attempt
				if attempt == opts.MaxAttempts {
					break
				}

				// Compute exponential backoff
				backoff := float64(opts.BaseBackoff) * pow(opts.BackoffMultiplier, float64(attempt-1))
				sleepDuration := time.Duration(backoff)

				// Context-aware sleep
				select {
				case <-time.After(sleepDuration):
				case <-ctx.Done():
					return
				}
			}

			// All attempts failed — fail-fast behavior
			select {
			case errCh <- lastErr:
				cancel()
			default:
			}

		}(jobID, fn)
	}

	// Close channels when all jobs finish
	go func() {
		wg.Wait()
		close(errCh)
	}()

	// Return first error, or nil if all succeeded
	return <-errCh
}

// RunWithRetry executes a single function with retry logic, exponential backoff,
// and context timeout.
func RunWithRetry(
	ctx context.Context,
	opts RetryOptions,
	fn func(ctx context.Context) error,
) error {
	// 1. Apply defaults
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = time.Second
	}
	if opts.BackoffMultiplier <= 0 {
		opts.BackoffMultiplier = 1.5
	}
	if opts.AttemptTimeout <= 0 {
		opts.AttemptTimeout = 20 * time.Second
	}

	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {

		// 2. Execute the function with Panic Recovery and Timeout
		// We wrap this in an anonymous function to handle defer/recover cleanly per attempt.
		err := func() (execErr error) {
			defer func() {
				if r := recover(); r != nil {
					execErr = fmt.Errorf("panic in retryable function: %v", r)
				}
			}()

			attemptCtx, attemptCancel := context.WithTimeout(ctx, opts.AttemptTimeout)
			defer attemptCancel()

			return fn(attemptCtx)
		}()

		// Success
		if err == nil {
			return nil
		}

		lastErr = err

		// If panic occurred, we treat it as a fatal error.
		if isPanicError(err) {
			return err
		}

		if attempt == opts.MaxAttempts {
			break
		}

		// 3. Compute exponential backoff
		backoff := float64(opts.BaseBackoff) * pow(opts.BackoffMultiplier, float64(attempt-1))
		sleepDuration := time.Duration(backoff)

		// 4. Context-aware sleep
		select {
		case <-time.After(sleepDuration):
			// Continue to next attempt
		case <-ctx.Done():
			// Parent context canceled, return the context error (or lastErr)
			return ctx.Err()
		}
	}

	return lastErr
}

// Helper to detect if the error came from panic recovery
func isPanicError(err error) bool {
	return err != nil && len(err.Error()) >= 5 && err.Error()[:5] == "panic"
}

func pow(x, y float64) float64 {
	r := 1.0
	for i := 0; i < int(y); i++ {
		r *= x
	}
	return r
}

// WorkerPool provides a pool of workers that process tasks submitted via channel.
// This allows dynamic task submission at runtime, unlike RunParallel which requires
// all tasks upfront. Useful for scenarios where tasks are discovered during processing.
//
// Usage:
//
//	pool := NewWorkerPool(ctx, 5) // 5 concurrent workers
//	pool.Start()
//
//	// Submit tasks dynamically
//	pool.Submit(func(ctx context.Context) error { ... })
//	pool.Submit(func(ctx context.Context) error { ... })
//
//	// When done submitting
//	pool.Close()       // Stop accepting new tasks
//	pool.Wait()        // Wait for all tasks to complete
//	errors := pool.Errors() // Get all errors
type WorkerPool struct {
	ctx         context.Context
	cancel      context.CancelFunc
	taskCh      chan func(context.Context) error
	wg          sync.WaitGroup
	errMu       sync.Mutex
	errors      []error
	workerCount int
	closed      bool
	closeMu     sync.Mutex
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
// The pool does not start processing until Start() is called.
func NewWorkerPool(ctx context.Context, workerCount int) *WorkerPool {
	if workerCount <= 0 {
		workerCount = 1
	}

	poolCtx, cancel := context.WithCancel(ctx)

	return &WorkerPool{
		ctx:         poolCtx,
		cancel:      cancel,
		taskCh:      make(chan func(context.Context) error, workerCount*2), // Buffered channel
		workerCount: workerCount,
		errors:      make([]error, 0),
	}
}

// Start launches the worker goroutines. Call this before submitting tasks.
func (p *WorkerPool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the goroutine that processes tasks from the channel
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.taskCh:
			if !ok {
				return // Channel closed
			}
			p.executeTask(task)
		}
	}
}

// executeTask runs a single task with panic recovery
func (p *WorkerPool) executeTask(task func(context.Context) error) {
	defer func() {
		if r := recover(); r != nil {
			p.addError(fmt.Errorf("panic in worker pool task: %v", r))
			zap.L().Error("Panic recovered in worker pool task", zap.Any("panic", r))
		}
	}()

	if err := task(p.ctx); err != nil {
		p.addError(err)
	}
}

// addError safely appends an error to the error list
func (p *WorkerPool) addError(err error) {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	p.errors = append(p.errors, err)
}

// Submit adds a task to the pool. Returns false if the pool is closed or context cancelled.
// This method is safe for concurrent use.
func (p *WorkerPool) Submit(task func(context.Context) error) bool {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return false
	}
	p.closeMu.Unlock()

	select {
	case <-p.ctx.Done():
		return false
	case p.taskCh <- task:
		return true
	}
}

// SubmitWait adds a task and blocks until it's accepted (useful when channel is full).
// Returns false only if context is cancelled or pool is closed.
func (p *WorkerPool) SubmitWait(task func(context.Context) error) bool {
	p.closeMu.Lock()
	if p.closed {
		p.closeMu.Unlock()
		return false
	}
	p.closeMu.Unlock()

	select {
	case <-p.ctx.Done():
		return false
	case p.taskCh <- task:
		return true
	}
}

// Close stops accepting new tasks. Existing tasks will continue processing.
// Call Wait() after Close() to wait for all tasks to complete.
func (p *WorkerPool) Close() {
	p.closeMu.Lock()
	defer p.closeMu.Unlock()

	if !p.closed {
		p.closed = true
		close(p.taskCh)
	}
}

// Wait blocks until all submitted tasks have completed.
// Should be called after Close().
func (p *WorkerPool) Wait() {
	p.wg.Wait()
}

// Cancel stops all workers immediately, abandoning pending tasks.
func (p *WorkerPool) Cancel() {
	p.cancel()
	p.Close()
}

// Errors returns all errors collected from task executions.
// Safe to call after Wait() returns.
func (p *WorkerPool) Errors() []error {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return append([]error(nil), p.errors...) // Return a copy
}

// HasErrors returns true if any task returned an error.
func (p *WorkerPool) HasErrors() bool {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return len(p.errors) > 0
}

// ErrorCount returns the number of errors collected.
func (p *WorkerPool) ErrorCount() int {
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return len(p.errors)
}
