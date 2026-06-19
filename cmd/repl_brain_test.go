package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
)

func TestCmdBrainStatusEmpty(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdBrainStatus()

	out := buf.String()
	if !strings.Contains(out, "No brain") || !strings.Contains(out, "train") {
		t.Errorf("empty brain-status should show hint, got: %q", out)
	}
}

func TestCmdBrainStatusWithMetrics(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID:             "test-user",
		TotalPredictions:   200,
		CorrectPredictions: 170,
		UserCorrections:    30,
		TrainingSize:       500,
		Accuracy:           0.85,
		LastTrainedAt:      time.Now().Add(-2 * time.Hour),
	})

	state.cmdBrainStatus()

	out := buf.String()
	if !strings.Contains(out, "85") {
		t.Errorf("brain-status should show accuracy, got: %q", out)
	}
	if !strings.Contains(out, "200") {
		t.Errorf("brain-status should show total predictions, got: %q", out)
	}
	if !strings.Contains(out, "500") {
		t.Errorf("brain-status should show training size, got: %q", out)
	}
}

func TestCmdRecategorize(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateTransaction(ctx, &models.Transaction{
		UserID:          "test-user",
		Type:            "debit",
		Amount:          450,
		Description:     "UPI-swiggy",
		Category:        "Uncategorized",
		ReviewStatus:    "pending_review",
		TransactionDate: time.Now(),
		Source:          "gmail_alert",
	})

	state.db.SeedDefaultCategories(ctx, "test-user")

	state.cmdRecategorize()

	out := buf.String()
	if !strings.Contains(out, "Recategoriz") {
		t.Errorf("recategorize should show progress, got: %q", out)
	}
}

func TestDispatchBrainStatusCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("brain-status")
	if result != Continue {
		t.Error("brain-status command should return Continue")
	}
}

func TestDispatchRecategorizeCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("recategorize")
	if result != Continue {
		t.Error("recategorize command should return Continue")
	}
}
