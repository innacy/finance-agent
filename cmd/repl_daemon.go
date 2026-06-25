package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/innacy/finance-agent/pkg/daemon"
)

func (s *replState) cmdDaemonStart() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	if s.daemon != nil && s.daemon.IsRunning() {
		s.printer.Warn("Daemon is already running. Use 'daemon-status' to check.")
		return
	}

	syncInterval := 5 * time.Minute
	if s.cfg != nil && s.cfg.Daemon.PollInterval > 0 {
		syncInterval = s.cfg.Daemon.PollInterval
	}

	syncJob := func(ctx context.Context) error {
		return nil
	}

	s.daemon = daemon.New(daemon.Config{
		SyncInterval: syncInterval,
		SyncJob:      syncJob,
	})

	s.daemon.Start()
	s.printer.Success(fmt.Sprintf("Daemon started — syncing every %s", syncInterval))
}

func (s *replState) cmdDaemonStop() {
	if s.daemon == nil || !s.daemon.IsRunning() {
		s.printer.Warn("Daemon is not running.")
		return
	}

	s.daemon.Stop()
	s.printer.Success("Daemon stopped.")
}

func (s *replState) cmdDaemonStatus() {
	if s.daemon == nil || !s.daemon.IsRunning() {
		s.printer.Info("Daemon is not running. Use 'daemon-start' to begin background sync.")
		return
	}

	status := s.daemon.Status()

	rows := [][]string{
		{"Status", "Running"},
		{"Started", formatTimeAgo(status.StartedAt)},
		{"Sync Runs", fmt.Sprintf("%d", status.SyncCount)},
	}
	if status.LastError != "" {
		rows = append(rows, []string{"Last Error", status.LastError})
	}

	s.printer.Box("Daemon Status", "")
	s.printer.Table([]string{"Field", "Value"}, rows)
}
