package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
)

func TestCmdSyncNoAuth(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdSync()

	out := buf.String()
	if !strings.Contains(out, "gmail-auth") {
		t.Errorf("sync without auth should mention gmail-auth, got: %q", out)
	}
}

func TestCmdSyncStatus(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdSyncStatus()

	out := buf.String()
	if !strings.Contains(out, "No sync") {
		t.Errorf("empty sync status should show 'No sync', got: %q", out)
	}
}

func TestCmdSyncStatusWithState(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.UpsertSyncState(ctx, &models.SyncState{
		UserID:         "test-user",
		Source:         "gmail",
		LastSyncTime:   time.Now().Add(-1 * time.Hour),
		LastMessageID:  "msg-latest",
		TotalProcessed: 150,
		Status:         "idle",
	})

	state.cmdSyncStatus()

	out := buf.String()
	if !strings.Contains(out, "150") {
		t.Errorf("sync status should show total processed, got: %q", out)
	}
	if !strings.Contains(out, "idle") {
		t.Errorf("sync status should show status, got: %q", out)
	}
}

func TestCmdTxnsEmpty(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	state.cmdTxns()

	out := buf.String()
	if !strings.Contains(out, "No transactions") {
		t.Errorf("empty txns should show 'No transactions', got: %q", out)
	}
}

func TestCmdTxnsShowsList(t *testing.T) {
	state, buf, cleanup := newTestStateWithDB(t)
	defer cleanup()

	ctx := context.Background()
	state.db.CreateTransaction(ctx, &models.Transaction{
		UserID:          "test-user",
		Type:            "debit",
		Amount:          450.00,
		Description:     "UPI-swiggy@axisbank",
		Merchant:        "SWIGGY",
		TransactionDate: time.Now(),
		Source:          "gmail_alert",
	})
	state.db.CreateTransaction(ctx, &models.Transaction{
		UserID:          "test-user",
		Type:            "credit",
		Amount:          15000.00,
		Description:     "NEFT-SALARY",
		TransactionDate: time.Now(),
		Source:          "gmail_alert",
	})

	state.cmdTxns()

	out := buf.String()
	if !strings.Contains(out, "450") {
		t.Error("txns should show debit amount")
	}
	if !strings.Contains(out, "15,000") {
		t.Error("txns should show credit amount with comma")
	}
}

func TestDispatchSyncCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("sync")
	if result != Continue {
		t.Error("sync command should return Continue")
	}
}

func TestDispatchSyncStatusCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("sync-status")
	if result != Continue {
		t.Error("sync-status command should return Continue")
	}
}

func TestDispatchTxnsCommand(t *testing.T) {
	state, _, cleanup := newTestStateWithDB(t)
	defer cleanup()

	result := state.dispatch("txns")
	if result != Continue {
		t.Error("txns command should return Continue")
	}
}
