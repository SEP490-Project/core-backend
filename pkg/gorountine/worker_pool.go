// Package gorountine provides utilities for managing goroutines in Go.
package gorountine

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"
)

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
			zap.L().Debug("Worker starting task", zap.Int("worker_id", id))
			p.executeTask(task)
			zap.L().Debug("Worker completed task", zap.Int("worker_id", id))
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
