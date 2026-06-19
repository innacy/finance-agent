package cmd

import (
	"context"
	"fmt"
)

func (s *replState) cmdCategories() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	cats, err := s.db.GetCategories(ctx, s.userID)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch categories: %v", err))
		return
	}

	if len(cats) == 0 {
		s.printer.Info("No categories found. They will be seeded on first sync.")
		return
	}

	s.printer.Box("Categories", "")

	rows := make([][]string, 0, len(cats))
	for _, cat := range cats {
		kwCount := fmt.Sprintf("%d keywords", len(cat.Keywords))
		defaultStr := ""
		if cat.IsDefault {
			defaultStr = "default"
		}
		rows = append(rows, []string{cat.Icon, cat.Name, kwCount, defaultStr})
	}
	s.printer.Table([]string{"", "Name", "Keywords", "Type"}, rows)
}

func (s *replState) cmdReview() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	txns, err := s.db.GetPendingReviewTransactions(ctx, s.userID, 20)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch review queue: %v", err))
		return
	}

	if len(txns) == 0 {
		s.printer.Info("No transactions pending review. All categorized!")
		return
	}

	s.printer.Box(fmt.Sprintf("Review Queue (%d pending)", len(txns)), "")

	rows := make([][]string, 0, len(txns))
	for _, txn := range txns {
		amount := fmt.Sprintf("₹%s", formatAmount(txn.Amount))
		date := txn.TransactionDate.Format("02 Jan")
		desc := txn.Description
		if txn.Merchant != "" {
			desc = txn.Merchant
		}
		if len(desc) > 25 {
			desc = desc[:25] + "…"
		}
		rows = append(rows, []string{date, desc, amount, txn.Category})
	}
	s.printer.Table([]string{"Date", "Description", "Amount", "Category"}, rows)
}
