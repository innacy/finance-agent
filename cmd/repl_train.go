package cmd

import (
	"context"
	"fmt"

	"github.com/innacy/finance-agent/pkg/brain"
	"github.com/innacy/finance-agent/pkg/categorizer"
)

func (s *replState) cmdTrain() {
	if s.db == nil {
		s.printer.Warn("Not connected to database. Run 'start' first.")
		return
	}

	ctx := context.Background()
	txns, err := s.db.GetPendingReviewTransactions(ctx, s.userID, 10)
	if err != nil {
		s.printer.Error(fmt.Sprintf("Failed to fetch transactions: %v", err))
		return
	}

	if len(txns) == 0 {
		s.printer.Info("No transactions to train on. All categorized!")
		return
	}

	classifier := brain.NewClassifier()
	cat := categorizer.New(s.db, s.userID, 0.5)

	s.printer.Box(fmt.Sprintf("Training Mode — %d transactions", len(txns)), "")
	s.printer.Info("For each transaction, confirm (y) or provide correct category.")
	fmt.Fprintln(s.output)

	trainer := brain.NewTrainer(s.db, classifier, s.userID)

	for i, txn := range txns {
		merchant := txn.Merchant
		if merchant == "" && txn.CounterpartyUPI != "" {
			merchant = txn.CounterpartyUPI
		}

		predicted := cat.Categorize(ctx, &categorizer.CategorizeInput{
			Merchant:    merchant,
			Description: txn.Description,
			Channel:     txn.Channel,
			Type:        txn.Type,
			Amount:      txn.Amount,
		})

		desc := txn.Description
		if merchant != "" {
			desc = merchant
		}
		if len(desc) > 30 {
			desc = desc[:30] + "…"
		}

		s.printer.Info(fmt.Sprintf("[%d/%d] ₹%s — %s", i+1, len(txns), formatAmount(txn.Amount), desc))
		s.printer.Info(fmt.Sprintf("  AI suggests: %s (%.0f%% confidence)", predicted.Category, predicted.Confidence*100))

		text := fmt.Sprintf("%s %s %s", merchant, txn.Description, txn.Channel)
		trainer.RecordFeedback(ctx, brain.FeedbackInput{
			Text:              text,
			Merchant:          merchant,
			PredictedCategory: predicted.Category,
			ActualCategory:    predicted.Category,
			IsCorrection:      false,
		})
	}

	s.printer.Success(fmt.Sprintf("Trained on %d transactions", len(txns)))
}
