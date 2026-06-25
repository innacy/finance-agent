package daemon

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Config struct {
	SyncInterval time.Duration
	SyncJob      Job
}

type Status struct {
	Running   bool
	StartedAt time.Time
	SyncCount int64
	LastError string
}

type Daemon struct {
	cfg       Config
	running   atomic.Bool
	startedAt time.Time
	syncCount atomic.Int64
	lastError atomic.Value

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
}

func New(cfg Config) *Daemon {
	return &Daemon{cfg: cfg}
}

func (d *Daemon) Start() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running.Load() {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	d.running.Store(true)
	d.startedAt = time.Now()

	wrappedJob := func(ctx context.Context) error {
		err := d.cfg.SyncJob(ctx)
		d.syncCount.Add(1)
		if err != nil {
			d.lastError.Store(err.Error())
		}
		return err
	}

	scheduler := NewScheduler(d.cfg.SyncInterval, wrappedJob)

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		scheduler.Start(ctx)
	}()
}

func (d *Daemon) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running.Load() {
		return
	}

	d.cancel()
	d.wg.Wait()
	d.running.Store(false)
}

func (d *Daemon) IsRunning() bool {
	return d.running.Load()
}

func (d *Daemon) Status() Status {
	s := Status{
		Running:   d.running.Load(),
		StartedAt: d.startedAt,
		SyncCount: d.syncCount.Load(),
	}

	if v := d.lastError.Load(); v != nil {
		s.LastError = v.(string)
	}

	return s
}
