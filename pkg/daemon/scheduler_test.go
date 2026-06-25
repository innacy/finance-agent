package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsJob(t *testing.T) {
	var count int32
	job := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	s := NewScheduler(50*time.Millisecond, job)
	ctx, cancel := context.WithCancel(context.Background())

	go s.Start(ctx)
	time.Sleep(180 * time.Millisecond)
	cancel()

	got := atomic.LoadInt32(&count)
	if got < 2 {
		t.Errorf("expected at least 2 runs in 180ms with 50ms interval, got %d", got)
	}
}

func TestSchedulerStopsOnCancel(t *testing.T) {
	var count int32
	job := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	s := NewScheduler(50*time.Millisecond, job)
	ctx, cancel := context.WithCancel(context.Background())

	go s.Start(ctx)
	time.Sleep(80 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	countAfterCancel := atomic.LoadInt32(&count)
	time.Sleep(100 * time.Millisecond)
	countLater := atomic.LoadInt32(&count)

	if countLater != countAfterCancel {
		t.Errorf("scheduler should stop after cancel, count grew from %d to %d", countAfterCancel, countLater)
	}
}

func TestSchedulerHandlesJobError(t *testing.T) {
	var count int32
	job := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return context.DeadlineExceeded
	}

	s := NewScheduler(50*time.Millisecond, job)
	ctx, cancel := context.WithCancel(context.Background())

	go s.Start(ctx)
	time.Sleep(180 * time.Millisecond)
	cancel()

	got := atomic.LoadInt32(&count)
	if got < 2 {
		t.Errorf("scheduler should continue running after job error, got %d runs", got)
	}
}

func TestSchedulerRunsImmediately(t *testing.T) {
	var count int32
	job := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	s := NewScheduler(10*time.Second, job)
	ctx, cancel := context.WithCancel(context.Background())

	go s.Start(ctx)
	time.Sleep(50 * time.Millisecond)
	cancel()

	got := atomic.LoadInt32(&count)
	if got != 1 {
		t.Errorf("scheduler should run immediately on start, got %d", got)
	}
}
