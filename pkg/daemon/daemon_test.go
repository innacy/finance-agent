package daemon

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDaemonStartStop(t *testing.T) {
	var syncCount int32
	syncJob := func(ctx context.Context) error {
		atomic.AddInt32(&syncCount, 1)
		return nil
	}

	d := New(Config{
		SyncInterval: 50 * time.Millisecond,
		SyncJob:      syncJob,
	})

	if d.IsRunning() {
		t.Error("daemon should not be running initially")
	}

	d.Start()
	time.Sleep(30 * time.Millisecond)

	if !d.IsRunning() {
		t.Error("daemon should be running after Start()")
	}

	time.Sleep(150 * time.Millisecond)
	d.Stop()

	if d.IsRunning() {
		t.Error("daemon should not be running after Stop()")
	}

	got := atomic.LoadInt32(&syncCount)
	if got < 2 {
		t.Errorf("expected at least 2 sync runs, got %d", got)
	}
}

func TestDaemonDoubleStartNoOp(t *testing.T) {
	syncJob := func(ctx context.Context) error { return nil }

	d := New(Config{
		SyncInterval: time.Second,
		SyncJob:      syncJob,
	})

	d.Start()
	d.Start() // should not panic or create duplicate goroutines
	time.Sleep(30 * time.Millisecond)
	d.Stop()
}

func TestDaemonStopWithoutStartNoOp(t *testing.T) {
	syncJob := func(ctx context.Context) error { return nil }

	d := New(Config{
		SyncInterval: time.Second,
		SyncJob:      syncJob,
	})

	d.Stop() // should not panic
}

func TestDaemonStatus(t *testing.T) {
	var count int32
	syncJob := func(ctx context.Context) error {
		atomic.AddInt32(&count, 1)
		return nil
	}

	d := New(Config{
		SyncInterval: 50 * time.Millisecond,
		SyncJob:      syncJob,
	})

	status := d.Status()
	if status.Running {
		t.Error("status should show not running")
	}

	d.Start()
	time.Sleep(130 * time.Millisecond)

	status = d.Status()
	if !status.Running {
		t.Error("status should show running")
	}
	if status.SyncCount < 2 {
		t.Errorf("expected sync count >= 2, got %d", status.SyncCount)
	}
	if status.StartedAt.IsZero() {
		t.Error("started_at should be set")
	}

	d.Stop()
}

func TestDaemonGracefulShutdown(t *testing.T) {
	syncJob := func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
			return nil
		}
	}

	d := New(Config{
		SyncInterval: 50 * time.Millisecond,
		SyncJob:      syncJob,
	})

	d.Start()
	time.Sleep(30 * time.Millisecond)

	done := make(chan struct{})
	go func() {
		d.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("Stop() should return within 1s (graceful shutdown)")
	}
}
