package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
)

func TestCmdCategoriesEmpty(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdCategories()

	out := buf.String()
	if !strings.Contains(out, "No categories") {
		t.Errorf("empty categories should show hint, got: %q", out)
	}
}

func TestCmdCategoriesShowsList(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.SeedDefaultCategories(ctx, "test-user")

	state.cmdCategories()

	out := buf.String()
	if !strings.Contains(out, "Food & Dining") {
		t.Error("categories should show Food & Dining")
	}
	if !strings.Contains(out, "Transport") {
		t.Error("categories should show Transport")
	}
}

func TestCmdReviewEmpty(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdReview()

	out := buf.String()
	if !strings.Contains(out, "No transactions") || !strings.Contains(out, "review") {
		t.Errorf("empty review should show no pending, got: %q", out)
	}
}

func TestCmdReviewShowsPending(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateTransaction(ctx, &models.Transaction{
		UserID:          "test-user",
		Type:            "debit",
		Amount:          1200,
		Description:     "UPI-unknown@upi",
		Category:        "Uncategorized",
		ReviewStatus:    "pending_review",
		TransactionDate: time.Now(),
		Source:          "gmail_alert",
	})
	state.db.CreateTransaction(ctx, &models.Transaction{
		UserID:          "test-user",
		Type:            "debit",
		Amount:          450,
		Description:     "UPI-swiggy@axisbank",
		Category:        "Food & Dining",
		ReviewStatus:    "auto_accepted",
		TransactionDate: time.Now(),
		Source:          "gmail_alert",
	})

	state.cmdReview()

	out := buf.String()
	if !strings.Contains(out, "1,200") || !strings.Contains(out, "Uncategorized") {
		t.Errorf("review should show pending transaction, got: %q", out)
	}
}

func TestDispatchCategoriesCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("categories")
	if result != Continue {
		t.Error("categories command should return Continue")
	}
}

func TestDispatchReviewCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("review")
	if result != Continue {
		t.Error("review command should return Continue")
	}
}
