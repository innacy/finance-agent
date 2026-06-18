package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/innacy/finance-agent/pkg/config"
	"github.com/innacy/finance-agent/pkg/db"
	"github.com/innacy/finance-agent/pkg/output"
)

type DispatchResult int

const (
	Continue DispatchResult = iota
	Exit
)

type replState struct {
	cfg     *config.Config
	printer *output.Printer
	output  io.Writer
	db      *db.Client
}

func RunREPL(cfg *config.Config) error {
	var buf bytes.Buffer
	w := io.MultiWriter(os.Stdout, &buf)
	printer := output.NewPrinter(w, true)

	state := &replState{
		cfg:     cfg,
		printer: printer,
		output:  os.Stdout,
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "finance-agent > ",
		HistoryFile:     "/tmp/finance-agent-history.tmp",
		AutoComplete:    buildCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return fmt.Errorf("initializing readline: %w", err)
	}
	defer rl.Close()

	printer.Info("finance-agent v0.1.0 — type 'help' for commands")
	fmt.Fprintln(state.output)

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}

		result := state.dispatch(line)
		if result == Exit {
			break
		}
	}

	return nil
}

func (s *replState) dispatch(input string) DispatchResult {
	input = strings.TrimSpace(input)
	if input == "" {
		return Continue
	}

	parts := strings.Fields(input)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "help":
		s.cmdHelp()
	case "config":
		s.cmdConfig()
	case "start":
		s.cmdStart()
	case "exit", "quit":
		s.cmdExit()
		return Exit
	default:
		s.printer.Warn(fmt.Sprintf("Unknown command: %q — type 'help' for available commands", cmd))
	}

	return Continue
}

func getCommandList() []string {
	return []string{
		"start", "config", "help", "exit",
		"accounts", "account-add", "account-update", "account-remove", "balance",
		"txns", "txn-search", "txn-categorize", "txn-tag", "txn-recurring",
		"cards", "card-add", "card-bill", "card-spend", "card-due",
		"gmail-auth", "sync", "sync-status", "sync-history", "import",
		"overview", "monthly", "spend", "spend-trend",
		"categories", "category-add", "category-edit", "category-remove", "recategorize",
		"brain-status", "review", "train", "brain-reset", "brain-export", "brain-import",
		"daemon-start", "daemon-stop", "daemon-status",
	}
}

func buildCompleter() *readline.PrefixCompleter {
	items := make([]readline.PrefixCompleterInterface, 0)
	for _, cmd := range getCommandList() {
		items = append(items, readline.PcItem(cmd))
	}
	return readline.NewPrefixCompleter(items...)
}
