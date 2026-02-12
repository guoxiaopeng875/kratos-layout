package job

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func TestTickerJob_StartStop(t *testing.T) {
	var count atomic.Int32
	j := newTickerJob("test-job", 50*time.Millisecond, log.DefaultLogger, func(_ context.Context) {
		count.Add(1)
	}, false)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	time.Sleep(180 * time.Millisecond)
	if err := j.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("Start returned error: %v", err)
	}

	got := count.Load()
	if got < 2 {
		t.Errorf("expected at least 2 executions, got %d", got)
	}
}

func TestTickerJob_StopMultipleTimes(t *testing.T) {
	j := newTickerJob("test-job", time.Hour, log.DefaultLogger, func(_ context.Context) {}, false)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	// Give Start time to enter select loop
	time.Sleep(20 * time.Millisecond)

	// Call Stop multiple times — must not panic
	for i := 0; i < 5; i++ {
		if err := j.Stop(ctx); err != nil {
			t.Fatalf("Stop call %d returned error: %v", i, err)
		}
	}

	if err := <-done; err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
}

func TestTickerJob_ExecuteImmediate(t *testing.T) {
	var count atomic.Int32
	j := newTickerJob("test-job", time.Hour, log.DefaultLogger, func(_ context.Context) {
		count.Add(1)
	}, true)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	// Wait enough for the immediate execution but not the ticker
	time.Sleep(20 * time.Millisecond)

	if err := j.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	<-done

	if got := count.Load(); got != 1 {
		t.Errorf("expected exactly 1 immediate execution, got %d", got)
	}
}

func TestTickerJob_NoExecuteImmediate(t *testing.T) {
	var count atomic.Int32
	j := newTickerJob("test-job", time.Hour, log.DefaultLogger, func(_ context.Context) {
		count.Add(1)
	}, false)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	time.Sleep(20 * time.Millisecond)

	if err := j.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
	<-done

	if got := count.Load(); got != 0 {
		t.Errorf("expected 0 executions, got %d", got)
	}
}

func TestTickerJob_ContextCancellation(t *testing.T) {
	j := newTickerJob("test-job", time.Hour, log.DefaultLogger, func(_ context.Context) {}, false)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	time.Sleep(20 * time.Millisecond)
	cancel()

	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestTickerJob_StopWaitsForExecution(t *testing.T) {
	var started atomic.Bool
	var finished atomic.Bool
	execStarted := make(chan struct{})

	j := newTickerJob("test-job", 20*time.Millisecond, log.DefaultLogger, func(_ context.Context) {
		if started.CompareAndSwap(false, true) {
			close(execStarted)
			time.Sleep(80 * time.Millisecond) // Simulate long-running task
			finished.Store(true)
		}
	}, false)

	ctx := context.Background()
	done := make(chan error, 1)
	go func() { done <- j.Start(ctx) }()

	// Wait for first execution to start
	select {
	case <-execStarted:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("execution did not start in time")
	}

	// Verify execution started but not finished
	if !started.Load() {
		t.Fatal("execution should have started")
	}
	if finished.Load() {
		t.Fatal("execution should not have finished yet")
	}

	// Call Stop while execution is in progress
	if err := j.Stop(ctx); err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	// Wait for Start to return (it should wait for execution to complete)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
		// By the time Start returns, execution must have completed
		if !finished.Load() {
			t.Error("execution should have finished before Start returned")
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Start did not return in time")
	}
}
