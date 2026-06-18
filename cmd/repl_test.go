package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/innacy/finance-agent/pkg/config"
	"github.com/innacy/finance-agent/pkg/output"
)

func newTestState() (*replState, *bytes.Buffer) {
	var buf bytes.Buffer
	cfg, _ := config.Load("")
	printer := output.NewPrinter(&buf, false)
	state := &replState{
		cfg:     cfg,
		printer: printer,
		output:  &buf,
	}
	return state, &buf
}

func TestDispatchHelp(t *testing.T) {
	state, buf := newTestState()

	result := state.dispatch("help")

	if result != Continue {
		t.Errorf("help should return Continue, got %v", result)
	}
	out := buf.String()
	if !strings.Contains(out, "accounts") {
		t.Error("help should list 'accounts' command")
	}
	if !strings.Contains(out, "sync") {
		t.Error("help should list 'sync' command")
	}
	if !strings.Contains(out, "exit") {
		t.Error("help should list 'exit' command")
	}
}

func TestDispatchConfig(t *testing.T) {
	state, buf := newTestState()

	result := state.dispatch("config")

	if result != Continue {
		t.Errorf("config should return Continue, got %v", result)
	}
	out := buf.String()
	if !strings.Contains(out, "mongodb://localhost:27017") {
		t.Error("config should show DB URI")
	}
	if !strings.Contains(out, "finance-agent") {
		t.Error("config should show DB name")
	}
}

func TestDispatchExit(t *testing.T) {
	state, _ := newTestState()

	result := state.dispatch("exit")

	if result != Exit {
		t.Errorf("exit should return Exit, got %v", result)
	}
}

func TestDispatchEmpty(t *testing.T) {
	state, _ := newTestState()

	result := state.dispatch("")

	if result != Continue {
		t.Error("empty input should return Continue")
	}
}

func TestDispatchUnknownCommand(t *testing.T) {
	state, buf := newTestState()

	result := state.dispatch("nonexistent-command")

	if result != Continue {
		t.Error("unknown command should return Continue")
	}
	out := buf.String()
	if !strings.Contains(out, "unknown command") && !strings.Contains(out, "Unknown") {
		t.Errorf("should show unknown command message, got: %q", out)
	}
}

func TestDispatchTrimsWhitespace(t *testing.T) {
	state, _ := newTestState()

	result := state.dispatch("  exit  ")

	if result != Exit {
		t.Error("dispatch should trim whitespace before matching")
	}
}

func TestDispatchCaseInsensitive(t *testing.T) {
	state, _ := newTestState()

	result := state.dispatch("EXIT")

	if result != Exit {
		t.Error("dispatch should be case-insensitive")
	}
}

func TestCommandListComplete(t *testing.T) {
	commands := getCommandList()

	required := []string{"start", "config", "help", "exit"}
	for _, cmd := range required {
		found := false
		for _, c := range commands {
			if c == cmd {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("command list should contain %q", cmd)
		}
	}
}
