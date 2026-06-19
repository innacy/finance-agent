package cmd

import (
	"context"
	"fmt"
	"time"
)

func (s *replState) cmdSync() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	if !s.gmailAuthed {
		s.printer.Warn("Gmail not authenticated. Run 'gmail-auth' first to connect your email.")
		return
	}

	s.printer.Info("Starting email sync...")
	// Actual sync logic will use the pipeline once gmail auth is in place
	s.printer.Success("Sync complete")
}

func (s *replState) cmdSyncStatus() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	state, err := s.db.GetSyncState(ctx, s.userID, "gmail")
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to get sync state: %v", err))
		return
	}

	if state == nil {
		s.printer.Info("No sync history found. Run 'sync' to start.")
		return
	}

	rows := [][]string{
		{"Status", state.Status},
		{"Total Processed", fmt.Sprintf("%d", state.TotalProcessed)},
		{"Last Sync", formatTimeAgo(state.LastSyncTime)},
		{"Last Message", state.LastMessageID},
	}
	if state.LastError != "" {
		rows = append(rows, []string{"Last Error", state.LastError})
	}

	s.printer.Box("Sync Status — Gmail", "")
	s.printer.Table([]string{"Field", "Value"}, rows)
}

func (s *replState) cmdTxns() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	txns, err := s.db.GetTransactionsByUser(ctx, s.userID, 30, 0, 20)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch transactions: %v", err))
		return
	}

	if len(txns) == 0 {
		s.printer.Info("No transactions found. Run 'sync' to fetch from email.")
		return
	}

	s.printer.Box("Recent Transactions (30 days)", "")

	rows := make([][]string, 0, len(txns))
	for _, txn := range txns {
		sign := "+"
		if txn.Type == "debit" {
			sign = "-"
		}
		amount := fmt.Sprintf("%s₹%s", sign, formatAmount(txn.Amount))
		date := txn.TransactionDate.Format("02 Jan")
		desc := txn.Description
		if txn.Merchant != "" {
			desc = txn.Merchant
		}
		if len(desc) > 25 {
			desc = desc[:25] + "…"
		}

		rows = append(rows, []string{date, desc, amount, txn.Channel})
	}

	s.printer.Table([]string{"Date", "Description", "Amount", "Channel"}, rows)
}

func (s *replState) cmdGmailAuth() {
	s.printer.Info("Gmail OAuth authentication")
	s.printer.Info("This will open your browser for Google authentication.")
	s.printer.Warn("Make sure 'credentials.json' exists in the project root.")

	// Actual OAuth flow will be triggered here
	s.printer.Info("Run 'sync' after authentication to start fetching transactions.")
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}
