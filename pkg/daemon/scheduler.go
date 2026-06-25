package daemon

import (
	"context"
	"time"
)

type Job func(ctx context.Context) error

type Scheduler struct {
	interval time.Duration
	job      Job
}

func NewScheduler(interval time.Duration, job Job) *Scheduler {
	return &Scheduler{interval: interval, job: job}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.job(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.job(ctx)
		}
	}
}
