package db

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestCreateTransaction(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	txn := &models.Transaction{
		UserID:          "user1",
		AccountID:       bson.NewObjectID(),
		Type:            "debit",
		Amount:          450.00,
		BalanceAfter:    99550.00,
		Description:     "UPI-SWIGGY-swiggy@axisbank",
		Merchant:        "SWIGGY",
		Channel:         "UPI",
		CounterpartyUPI: "swiggy@axisbank",
		Source:          "gmail_alert",
		SourceRef:       "msg-123",
		TransactionDate: time.Now(),
		ReviewStatus:    "auto_accepted",
		Confidence:      0.95,
	}

	err := client.CreateTransaction(context.Background(), txn)
	if err != nil {
		t.Fatalf("CreateTransaction failed: %v", err)
	}
	if txn.ID.IsZero() {
		t.Error("expected ID to be set")
	}
	if txn.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestGetTransactionsByUser(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	for i := 0; i < 5; i++ {
		client.CreateTransaction(ctx, &models.Transaction{
			UserID:          "user1",
			AccountID:       bson.NewObjectID(),
			Type:            "debit",
			Amount:          float64(100 * (i + 1)),
			Description:     "TEST-TXN",
			TransactionDate: now.AddDate(0, 0, -i),
			Source:          "gmail_alert",
		})
	}
	client.CreateTransaction(ctx, &models.Transaction{
		UserID:          "user2",
		Type:            "debit",
		Amount:          999,
		Description:     "OTHER-USER",
		TransactionDate: now,
		Source:          "gmail_alert",
	})

	txns, err := client.GetTransactionsByUser(ctx, "user1", 30, 0, 50)
	if err != nil {
		t.Fatalf("GetTransactionsByUser failed: %v", err)
	}
	if len(txns) != 5 {
		t.Errorf("expected 5 transactions for user1, got %d", len(txns))
	}
}

func TestGetTransactionsRespectsDaysLimit(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()

	client.CreateTransaction(ctx, &models.Transaction{
		UserID: "user1", Type: "debit", Amount: 100,
		Description: "RECENT", TransactionDate: now.AddDate(0, 0, -5),
		Source: "gmail_alert",
	})
	client.CreateTransaction(ctx, &models.Transaction{
		UserID: "user1", Type: "debit", Amount: 200,
		Description: "OLD", TransactionDate: now.AddDate(0, 0, -60),
		Source: "gmail_alert",
	})

	txns, _ := client.GetTransactionsByUser(ctx, "user1", 30, 0, 50)
	if len(txns) != 1 {
		t.Errorf("expected 1 transaction within 30 days, got %d", len(txns))
	}
}

func TestDeduplicateTransaction(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now()
	accID := bson.NewObjectID()

	txn1 := &models.Transaction{
		UserID: "user1", AccountID: accID, Type: "debit", Amount: 450,
		Reference: "UPI-REF-123", TransactionDate: now, Source: "gmail_alert",
		SourceRef: "msg-1",
	}
	err := client.CreateTransaction(ctx, txn1)
	if err != nil {
		t.Fatal(err)
	}

	exists, err := client.TransactionExists(ctx, "user1", accID, now, 450, "UPI-REF-123")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("existing transaction should be detected as duplicate")
	}

	notExists, _ := client.TransactionExists(ctx, "user1", accID, now, 450, "DIFFERENT-REF")
	if notExists {
		t.Error("different reference should not be duplicate")
	}
}

func TestGetSyncState(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	state, err := client.GetSyncState(ctx, "user1", "gmail")
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		t.Error("should return nil for non-existent sync state")
	}

	err = client.UpsertSyncState(ctx, &models.SyncState{
		UserID:         "user1",
		Source:         "gmail",
		LastSyncTime:   time.Now(),
		LastMessageID:  "msg-abc",
		TotalProcessed: 42,
		Status:         "idle",
	})
	if err != nil {
		t.Fatalf("UpsertSyncState failed: %v", err)
	}

	state, err = client.GetSyncState(ctx, "user1", "gmail")
	if err != nil {
		t.Fatal(err)
	}
	if state == nil {
		t.Fatal("expected sync state to exist")
	}
	if state.TotalProcessed != 42 {
		t.Errorf("expected 42 processed, got %d", state.TotalProcessed)
	}
	if state.LastMessageID != "msg-abc" {
		t.Errorf("expected msg-abc, got %q", state.LastMessageID)
	}
}
