package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestCmdUIStartNoDB(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()
	state.db = nil

	state.cmdUIStart()

	out := buf.String()
	if !strings.Contains(out, "start") {
		t.Errorf("ui without db should warn, got: %q", out)
	}
}

func TestCmdUIStart(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdUIStart()
	time.Sleep(50 * time.Millisecond)

	out := buf.String()
	if !strings.Contains(out, "8090") && !strings.Contains(out, "UI") {
		t.Errorf("ui-start should show port, got: %q", out)
	}

	if state.uiServer == nil {
		t.Error("uiServer should be set after start")
	}

	state.cmdUIStop()
}

func TestCmdUIStop(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdUIStart()
	time.Sleep(50 * time.Millisecond)
	buf.Reset()

	state.cmdUIStop()

	out := buf.String()
	if !strings.Contains(out, "stop") && !strings.Contains(out, "Stop") {
		t.Errorf("ui-stop should confirm, got: %q", out)
	}
}

func TestDispatchUICommands(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("ui-start")
	if result != Continue {
		t.Error("ui-start should return Continue")
	}
	time.Sleep(50 * time.Millisecond)

	result = state.dispatch("ui-stop")
	if result != Continue {
		t.Error("ui-stop should return Continue")
	}
}
