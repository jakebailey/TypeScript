package core

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestGoroutinePoolLimitsConcurrency(t *testing.T) {
	t.Parallel()

	pool := newGoroutinePool(2)
	release := make(chan struct{})
	started := make(chan struct{}, 3)
	task := func() {
		started <- struct{}{}
		<-release
	}

	pool.Queue(task)
	<-started
	pool.Queue(task)
	<-started

	queued := make(chan struct{})
	go func() {
		pool.Queue(task)
		close(queued)
	}()
	assertBlocked(t, queued, "third task was queued while all workers were busy")

	close(release)
	<-queued
	pool.Close()
}

func TestGoroutinePoolRun(t *testing.T) {
	t.Parallel()

	pool := newGoroutinePool(1)
	defer pool.Close()

	release := make(chan struct{})
	started := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		pool.Run(func() {
			close(started)
			<-release
		})
		close(returned)
	}()

	<-started
	assertBlocked(t, returned, "Run returned before the task completed")
	close(release)
	<-returned
}

func TestGoroutinePoolRunPropagatesPanic(t *testing.T) {
	t.Parallel()

	pool := newGoroutinePool(1)
	defer pool.Close()

	const panicValue = "panic value"
	defer func() {
		if value := recover(); value != panicValue {
			t.Fatalf("recovered %v, want %q", value, panicValue)
		}
	}()
	pool.Run(func() {
		panic(panicValue)
	})
}

func TestImmediatePool(t *testing.T) {
	t.Parallel()

	pool := NewGOMAXPROCSPool(true)
	var calls atomic.Int32
	pool.Queue(func() {
		calls.Add(1)
	})
	pool.Run(func() {
		calls.Add(1)
	})
	if got := calls.Load(); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}

	pool.Close()
	assertPanic(t, func() {
		pool.Queue(func() {})
	})
	assertPanic(t, func() {
		pool.Run(func() {})
	})
}

func assertBlocked(t *testing.T, ch <-chan struct{}, message string) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal(message)
	case <-time.After(10 * time.Millisecond):
		return
	}
}

func assertPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("function did not panic")
		}
	}()
	fn()
}
