package cmd

import "fmt"

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
		s.printer.Success(fmt.Sprintf("Database: %s/%s", s.cfg.DB.URI, s.cfg.DB.Database))
	}
	s.printer.Warn("Gmail: not authenticated (run 'gmail-auth')")
	s.printer.Info("Type 'help' for available commands")
}

func (s *replState) cmdExit() {
	s.printer.Info("Session ended. Goodbye!")
}
