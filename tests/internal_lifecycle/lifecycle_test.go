package internal_lifecycle_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/JaimeStill/agent-lab/internal/lifecycle"
)

func TestNew(t *testing.T) {
	lc := lifecycle.New()

	if lc == nil {
		t.Fatal("New() returned nil")
	}

	if lc.Context() == nil {
		t.Error("Context() returned nil")
	}

	if lc.Ready() {
		t.Error("Ready() = true, want false for new coordinator")
	}
}

func TestCoordinator_Context(t *testing.T) {
	lc := lifecycle.New()
	ctx := lc.Context()

	select {
	case <-ctx.Done():
		t.Error("context should not be cancelled")
	default:
	}
}

func TestCoordinator_OnStartup(t *testing.T) {
	lc := lifecycle.New()

	var executed atomic.Bool
	lc.OnStartup(func() {
		executed.Store(true)
	})

	lc.WaitForStartup()

	if !executed.Load() {
		t.Error("startup function was not executed")
	}
}

func TestCoordinator_OnStartup_Multiple(t *testing.T) {
	lc := lifecycle.New()

	var count atomic.Int32
	for i := 0; i < 3; i++ {
		lc.OnStartup(func() {
			count.Add(1)
		})
	}

	lc.WaitForStartup()

	if count.Load() != 3 {
		t.Errorf("count = %d, want 3", count.Load())
	}
}

func TestCoordinator_WaitForStartup_SetsReady(t *testing.T) {
	lc := lifecycle.New()

	if lc.Ready() {
		t.Error("Ready() = true before WaitForStartup")
	}

	lc.WaitForStartup()

	if !lc.Ready() {
		t.Error("Ready() = false after WaitForStartup")
	}
}

func TestCoordinator_OnShutdown(t *testing.T) {
	lc := lifecycle.New()

	var executed atomic.Bool
	lc.OnShutdown(func() {
		<-lc.Context().Done()
		executed.Store(true)
	})

	err := lc.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	if !executed.Load() {
		t.Error("shutdown function was not executed")
	}
}

func TestCoordinator_OnShutdown_Multiple(t *testing.T) {
	lc := lifecycle.New()

	var count atomic.Int32
	for i := 0; i < 3; i++ {
		lc.OnShutdown(func() {
			<-lc.Context().Done()
			count.Add(1)
		})
	}

	err := lc.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	if count.Load() != 3 {
		t.Errorf("count = %d, want 3", count.Load())
	}
}

func TestCoordinator_Shutdown_CancelsContext(t *testing.T) {
	lc := lifecycle.New()
	ctx := lc.Context()

	err := lc.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	select {
	case <-ctx.Done():
	default:
		t.Error("context should be cancelled after shutdown")
	}
}

func TestCoordinator_Shutdown_Timeout(t *testing.T) {
	lc := lifecycle.New()

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		time.Sleep(500 * time.Millisecond)
	})

	err := lc.Shutdown(50 * time.Millisecond)
	if err == nil {
		t.Error("Shutdown() should return timeout error")
	}
}

func TestCoordinator_ReadinessChecker(t *testing.T) {
	lc := lifecycle.New()

	var checker lifecycle.ReadinessChecker = lc

	if checker.Ready() {
		t.Error("Ready() = true, want false")
	}

	lc.WaitForStartup()

	if !checker.Ready() {
		t.Error("Ready() = false, want true")
	}
}

func TestCoordinator_ConcurrentReady(t *testing.T) {
	lc := lifecycle.New()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			_ = lc.Ready()
		}
		close(done)
	}()

	lc.WaitForStartup()

	<-done
}

func TestCoordinator_FullLifecycle(t *testing.T) {
	lc := lifecycle.New()

	var startupComplete atomic.Bool
	var shutdownComplete atomic.Bool

	lc.OnStartup(func() {
		time.Sleep(10 * time.Millisecond)
		startupComplete.Store(true)
	})

	lc.OnShutdown(func() {
		<-lc.Context().Done()
		time.Sleep(10 * time.Millisecond)
		shutdownComplete.Store(true)
	})

	go lc.WaitForStartup()

	time.Sleep(50 * time.Millisecond)

	if !lc.Ready() {
		t.Error("Ready() = false after startup")
	}

	if !startupComplete.Load() {
		t.Error("startup did not complete")
	}

	err := lc.Shutdown(5 * time.Second)
	if err != nil {
		t.Fatalf("Shutdown() failed: %v", err)
	}

	if !shutdownComplete.Load() {
		t.Error("shutdown did not complete")
	}
}
