package cmd

import (
	"context"
	"fmt"

	"github.com/innacy/finance-agent/pkg/db"
)

func (s *replState) cmdHelp() {
	headers := []string{"Command", "Description"}
	rows := [][]string{
		{"start", "Initialize session, connect to DB"},
		{"config", "Show current configuration"},
		{"help", "Show this help message"},
		{"exit", "End session"},
		{"", ""},
		{"accounts", "List all bank accounts with balances"},
		{"account-add", "Add a new bank account"},
		{"balance", "Quick total balance across all accounts"},
		{"overview", "All accounts summary, net position"},
		{"", ""},
		{"txns", "List recent transactions"},
		{"txn-search", "Search transactions"},
		{"txn-categorize", "Manually categorize a transaction"},
		{"spend", "Category-wise spend breakdown"},
		{"spend-trend", "Month-over-month comparison"},
		{"monthly", "Income vs spend for current month"},
		{"", ""},
		{"cards", "List credit cards"},
		{"card-bill", "Current billing details"},
		{"card-due", "Upcoming due dates"},
		{"", ""},
		{"gmail-auth", "Run Gmail OAuth flow"},
		{"sync", "Pull new emails, parse & store"},
		{"sync-status", "Last sync time and status"},
		{"import", "Import PDF/CSV statement"},
		{"", ""},
		{"categories", "List categories with counts"},
		{"category-add", "Add categorization rule"},
		{"recategorize", "Re-run engine on uncategorized"},
		{"", ""},
		{"brain-status", "Brain accuracy and stats"},
		{"review", "Review uncertain transactions"},
		{"train", "Training session on weak spots"},
		{"", ""},
		{"daemon-start", "Start background Gmail polling"},
		{"daemon-stop", "Stop background polling"},
		{"daemon-status", "Show daemon state"},
	}

	s.printer.Table(headers, rows)
}

func (s *replState) cmdStart() {
	s.printer.Info("Initializing session...")
	s.printer.Success("Configuration loaded")

	if s.cfg.DB.URI != "" {
		client, err := db.NewClient(&s.cfg.DB)
		if err != nil {
			s.printer.Error(fmt.Sprintf("Database connection failed: %v", err))
			return
		}

		ctx := context.Background()
		if err := client.Ping(ctx); err != nil {
			s.printer.Error(fmt.Sprintf("Database ping failed: %v", err))
			return
		}

		if err := client.EnsureIndexes(ctx); err != nil {
			s.printer.Warn(fmt.Sprintf("Index creation warning: %v", err))
		}

		s.db = client
		s.userID = "default"
		s.printer.Success(fmt.Sprintf("Database: %s/%s", s.cfg.DB.URI, s.cfg.DB.Database))
	}

	s.printer.Warn("Gmail: not authenticated (run 'gmail-auth')")

	if s.db != nil {
		accounts, _ := s.db.GetAccountsByUser(context.Background(), s.userID)
		s.printer.Info(fmt.Sprintf("%d accounts loaded", len(accounts)))
	}

	s.printer.Info("Type 'help' for available commands")
}

func (s *replState) cmdExit() {
	s.printer.Info("Session ended. Goodbye!")
}
