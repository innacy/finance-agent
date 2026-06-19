package cmd

import (
	"context"
	"fmt"

	"github.com/innacy/finance-agent/pkg/categorizer"
)

func (s *replState) cmdBrainStatus() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	metrics, err := s.db.GetBrainMetrics(ctx, s.userID)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to get brain metrics: %v", err))
		return
	}

	if metrics == nil {
		s.printer.Info("No brain metrics yet. Run 'train' or sync transactions to start learning.")
		return
	}

	accuracyPct := fmt.Sprintf("%.1f%%", metrics.Accuracy*100)

	rows := [][]string{
		{"Accuracy", accuracyPct},
		{"Total Predictions", fmt.Sprintf("%d", metrics.TotalPredictions)},
		{"Correct Predictions", fmt.Sprintf("%d", metrics.CorrectPredictions)},
		{"User Corrections", fmt.Sprintf("%d", metrics.UserCorrections)},
		{"Training Size", fmt.Sprintf("%d", metrics.TrainingSize)},
	}

	if !metrics.LastTrainedAt.IsZero() {
		rows = append(rows, []string{"Last Trained", formatTimeAgo(metrics.LastTrainedAt)})
	}

	s.printer.Box("Brain Status", "")
	s.printer.Table([]string{"Metric", "Value"}, rows)
}

func (s *replState) cmdRecategorize() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	txns, err := s.db.GetPendingReviewTransactions(ctx, s.userID, 100)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch transactions: %v", err))
		return
	}

	if len(txns) == 0 {
		s.printer.Info("No uncategorized transactions to recategorize.")
		return
	}

	cat := categorizer.New(s.db, s.userID, 0.7)
	recategorized := 0

	for _, txn := range txns {
		merchant := txn.Merchant
		if merchant == "" && txn.CounterpartyUPI != "" {
			merchant = txn.CounterpartyUPI
		}

		result := cat.Categorize(ctx, &categorizer.CategorizeInput{
			Merchant:    merchant,
			Description: txn.Description,
			Channel:     txn.Channel,
			Type:        txn.Type,
			Amount:      txn.Amount,
		})

		if result.Category != "Uncategorized" && result.Confidence >= 0.7 {
			recategorized++
		}
	}

	s.printer.Success(fmt.Sprintf("Recategorized %d/%d transactions", recategorized, len(txns)))
}
