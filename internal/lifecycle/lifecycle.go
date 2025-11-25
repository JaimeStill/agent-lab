// Package lifecycle provides application lifecycle coordination for startup and shutdown.
package lifecycle

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ReadinessChecker provides a simple interface for checking if a system is ready.
type ReadinessChecker interface {
	Ready() bool
}

// Coordinator manages application lifecycle including startup hooks, shutdown hooks,
// and readiness state. It provides a shared context that is cancelled during shutdown.
type Coordinator struct {
	ctx        context.Context
	cancel     context.CancelFunc
	startupWg  sync.WaitGroup
	shutdownWg sync.WaitGroup
	ready      bool
	readyMu    sync.RWMutex
}

// New creates a new Coordinator with an active context.
func New() *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Context returns the coordinator's context, which is cancelled during shutdown.
func (c *Coordinator) Context() context.Context {
	return c.ctx
}

// OnStartup registers a function to run concurrently during startup.
// All registered functions must complete before WaitForStartup returns.
func (c *Coordinator) OnStartup(fn func()) {
	c.startupWg.Go(fn)
}

// OnShutdown registers a function to run concurrently during shutdown.
// Functions should wait for Context().Done() before performing cleanup.
func (c *Coordinator) OnShutdown(fn func()) {
	c.shutdownWg.Go(fn)
}

// Ready returns true after WaitForStartup has completed.
func (c *Coordinator) Ready() bool {
	c.readyMu.RLock()
	defer c.readyMu.RUnlock()
	return c.ready
}

// WaitForStartup blocks until all startup hooks complete, then marks the coordinator as ready.
func (c *Coordinator) WaitForStartup() {
	c.startupWg.Wait()
	c.readyMu.Lock()
	c.ready = true
	c.readyMu.Unlock()
}

// Shutdown cancels the context and waits for all shutdown hooks to complete.
// Returns an error if shutdown does not complete within the timeout.
func (c *Coordinator) Shutdown(timeout time.Duration) error {
	c.cancel()

	done := make(chan struct{})
	go func() {
		c.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	}
}
