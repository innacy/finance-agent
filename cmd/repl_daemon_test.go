package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestCmdDaemonStartNoDB(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()
	state.db = nil

	state.cmdDaemonStart()

	out := buf.String()
	if !strings.Contains(out, "start") {
		t.Errorf("daemon-start without db should warn, got: %q", out)
	}
}

func TestCmdDaemonStart(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdDaemonStart()

	out := buf.String()
	if !strings.Contains(out, "Daemon started") && !strings.Contains(out, "started") {
		t.Errorf("daemon-start should confirm start, got: %q", out)
	}

	if state.daemon == nil || !state.daemon.IsRunning() {
		t.Error("daemon should be running after start")
	}

	state.daemon.Stop()
}

func TestCmdDaemonStop(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdDaemonStart()
	time.Sleep(30 * time.Millisecond)
	buf.Reset()

	state.cmdDaemonStop()

	out := buf.String()
	if !strings.Contains(out, "stop") && !strings.Contains(out, "Stop") {
		t.Errorf("daemon-stop should confirm stop, got: %q", out)
	}
}

func TestCmdDaemonStatusNotRunning(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdDaemonStatus()

	out := buf.String()
	if !strings.Contains(out, "not running") && !strings.Contains(out, "Not running") {
		t.Errorf("daemon-status should show not running, got: %q", out)
	}
}

func TestCmdDaemonStatusRunning(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdDaemonStart()
	time.Sleep(30 * time.Millisecond)
	buf.Reset()

	state.cmdDaemonStatus()

	out := buf.String()
	if !strings.Contains(out, "Running") && !strings.Contains(out, "running") {
		t.Errorf("daemon-status should show running, got: %q", out)
	}

	state.cmdDaemonStop()
}

func TestDispatchDaemonStartCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("daemon-start")
	if result != Continue {
		t.Error("daemon-start should return Continue")
	}
	if state.daemon != nil {
		state.daemon.Stop()
	}
}

func TestDispatchDaemonStatusCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("daemon-status")
	if result != Continue {
		t.Error("daemon-status should return Continue")
	}
}

func TestDispatchDaemonStopCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("daemon-stop")
	if result != Continue {
		t.Error("daemon-stop should return Continue")
	}
}
