package db

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/config"
)

func setupTestDB(t *testing.T) (*Client, func()) {
	t.Helper()
	cfg := &config.DBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "finance-agent-test",
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create test client: %v", err)
	}

	ctx := context.Background()
	if err := client.Ping(ctx); err != nil {
		t.Skipf("MongoDB not available, skipping integration test: %v", err)
	}

	cleanup := func() {
		client.DB().Drop(ctx)
		client.Disconnect(ctx)
	}

	return client, cleanup
}

func TestCreateAccount(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	acc := &models.BankAccount{
		UserID:        "test-user",
		BankName:      "HDFC",
		AccountNumber: "4521",
		AccountType:   "savings",
		Balance:       124350.00,
		Currency:      "INR",
		LastUpdated:   time.Now(),
		IsActive:      true,
		IsConfirmed:   true,
	}

	err := client.CreateAccount(context.Background(), acc)
	if err != nil {
		t.Fatalf("CreateAccount failed: %v", err)
	}
	if acc.ID.IsZero() {
		t.Error("expected ID to be set after create")
	}
}

func TestGetAccountsByUser(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.CreateAccount(ctx, &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "1111",
		AccountType: "savings", Balance: 10000, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})
	client.CreateAccount(ctx, &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "2222",
		AccountType: "current", Balance: 50000, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})
	client.CreateAccount(ctx, &models.BankAccount{
		UserID: "user2", BankName: "SBI", AccountNumber: "3333",
		AccountType: "savings", Balance: 20000, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	})

	accounts, err := client.GetAccountsByUser(ctx, "user1")
	if err != nil {
		t.Fatalf("GetAccountsByUser failed: %v", err)
	}
	if len(accounts) != 2 {
		t.Errorf("expected 2 accounts for user1, got %d", len(accounts))
	}
}

func TestUpdateBalance(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	acc := &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "4521",
		AccountType: "savings", Balance: 10000, Currency: "INR",
		IsActive: true, IsConfirmed: true,
	}
	client.CreateAccount(ctx, acc)

	err := client.UpdateBalance(ctx, acc.ID, 15000.50)
	if err != nil {
		t.Fatalf("UpdateBalance failed: %v", err)
	}

	accounts, _ := client.GetAccountsByUser(ctx, "user1")
	if len(accounts) == 0 {
		t.Fatal("no accounts found")
	}
	if accounts[0].Balance != 15000.50 {
		t.Errorf("expected balance 15000.50, got %f", accounts[0].Balance)
	}
}

func TestGetTotalBalance(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.CreateAccount(ctx, &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "1111",
		Balance: 10000, Currency: "INR", IsActive: true, IsConfirmed: true,
	})
	client.CreateAccount(ctx, &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "2222",
		Balance: 25000, Currency: "INR", IsActive: true, IsConfirmed: true,
	})

	total, err := client.GetTotalBalance(ctx, "user1")
	if err != nil {
		t.Fatalf("GetTotalBalance failed: %v", err)
	}
	if total != 35000 {
		t.Errorf("expected total 35000, got %f", total)
	}
}

func TestDeactivateAccount(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	acc := &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "4521",
		Balance: 10000, Currency: "INR", IsActive: true, IsConfirmed: true,
	}
	client.CreateAccount(ctx, acc)

	err := client.DeactivateAccount(ctx, acc.ID)
	if err != nil {
		t.Fatalf("DeactivateAccount failed: %v", err)
	}

	accounts, _ := client.GetAccountsByUser(ctx, "user1")
	if len(accounts) != 0 {
		t.Error("deactivated account should not appear in active list")
	}
}

func TestDuplicateAccountRejected(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	client.EnsureIndexes(ctx)

	acc := &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "4521",
		Balance: 10000, Currency: "INR", IsActive: true, IsConfirmed: true,
	}
	err := client.CreateAccount(ctx, acc)
	if err != nil {
		t.Fatalf("first create failed: %v", err)
	}

	dup := &models.BankAccount{
		UserID: "user1", BankName: "HDFC", AccountNumber: "4521",
		Balance: 20000, Currency: "INR", IsActive: true, IsConfirmed: true,
	}
	err = client.CreateAccount(ctx, dup)
	if err == nil {
		t.Error("expected error for duplicate account")
	}
}
