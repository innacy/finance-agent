package sync

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/gmail"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type mockDB struct {
	transactions []models.Transaction
	accounts     []models.BankAccount
	syncState    *models.SyncState
}

func (m *mockDB) CreateTransaction(ctx context.Context, txn *models.Transaction) error {
	txn.ID = bson.NewObjectID()
	txn.CreatedAt = time.Now()
	m.transactions = append(m.transactions, *txn)
	return nil
}

func (m *mockDB) TransactionExists(ctx context.Context, userID string, accountID bson.ObjectID, date time.Time, amount float64, reference string) (bool, error) {
	for _, t := range m.transactions {
		if t.UserID == userID && t.Amount == amount && t.Reference == reference {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockDB) GetAccountsByUser(ctx context.Context, userID string) ([]models.BankAccount, error) {
	var result []models.BankAccount
	for _, a := range m.accounts {
		if a.UserID == userID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockDB) CreateAccount(ctx context.Context, account *models.BankAccount) error {
	account.ID = bson.NewObjectID()
	m.accounts = append(m.accounts, *account)
	return nil
}

func (m *mockDB) UpdateBalance(ctx context.Context, id bson.ObjectID, balance float64) error {
	for i := range m.accounts {
		if m.accounts[i].ID == id {
			m.accounts[i].Balance = balance
		}
	}
	return nil
}

func (m *mockDB) GetSyncState(ctx context.Context, userID, source string) (*models.SyncState, error) {
	return m.syncState, nil
}

func (m *mockDB) UpsertSyncState(ctx context.Context, state *models.SyncState) error {
	m.syncState = state
	return nil
}

func TestPipelineProcessesNewTransactions(t *testing.T) {
	db := &mockDB{}
	db.accounts = []models.BankAccount{
		{ID: bson.NewObjectID(), UserID: "user1", AccountNumber: "4521", BankName: "HDFC", IsActive: true},
	}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-1",
			Subject: "Alert : Update for your HDFC Bank A/c XX4521",
			Body:    "Dear Customer,\nRs.450.00 has been debited from account **4521 on 15-06-2026 to VPA swiggy@axisbank (UPI Ref No. 415678901234).\nAvailable Balance: Rs.99,550.00",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipeline(db, "user1")
	result := p.Process(context.Background(), emails)

	if result.Processed != 1 {
		t.Errorf("expected 1 processed, got %d", result.Processed)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
	if result.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", result.Errors)
	}
	if len(db.transactions) != 1 {
		t.Fatalf("expected 1 transaction in db, got %d", len(db.transactions))
	}
	if db.transactions[0].Amount != 450.00 {
		t.Errorf("expected amount 450, got %f", db.transactions[0].Amount)
	}
}

func TestPipelineSkipsDuplicates(t *testing.T) {
	db := &mockDB{}
	db.accounts = []models.BankAccount{
		{ID: bson.NewObjectID(), UserID: "user1", AccountNumber: "4521", BankName: "HDFC", IsActive: true},
	}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-1",
			Subject: "Alert : Update for your HDFC Bank A/c XX4521",
			Body:    "Dear Customer,\nRs.450.00 has been debited from account **4521 on 15-06-2026 to VPA swiggy@axisbank (UPI Ref No. 415678901234).\nAvailable Balance: Rs.99,550.00",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipeline(db, "user1")
	p.Process(context.Background(), emails)
	result := p.Process(context.Background(), emails)

	if result.Duplicates != 1 {
		t.Errorf("expected 1 duplicate, got %d", result.Duplicates)
	}
	if result.Created != 0 {
		t.Errorf("expected 0 created on second pass, got %d", result.Created)
	}
}

func TestPipelineAutoDetectsNewAccount(t *testing.T) {
	db := &mockDB{}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-1",
			Subject: "Alert : Update for your HDFC Bank A/c XX9876",
			Body:    "Dear Customer,\nRs.5,000.00 has been debited from account **9876 on 15-06-2026 to VPA shop@upi (UPI Ref No. 999888777666).\nAvailable Balance: Rs.45,000.00",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipeline(db, "user1")
	result := p.Process(context.Background(), emails)

	if result.AccountsDetected != 1 {
		t.Errorf("expected 1 account detected, got %d", result.AccountsDetected)
	}
	if len(db.accounts) != 1 {
		t.Fatalf("expected 1 account in db, got %d", len(db.accounts))
	}
	if db.accounts[0].AccountNumber != "9876" {
		t.Errorf("expected account 9876, got %q", db.accounts[0].AccountNumber)
	}
	if db.accounts[0].IsConfirmed {
		t.Error("auto-detected account should not be confirmed")
	}
	if db.accounts[0].DetectedFrom != "gmail_alert" {
		t.Errorf("expected detected_from gmail_alert, got %q", db.accounts[0].DetectedFrom)
	}
}

func TestPipelineUpdatesBalance(t *testing.T) {
	accID := bson.NewObjectID()
	db := &mockDB{}
	db.accounts = []models.BankAccount{
		{ID: accID, UserID: "user1", AccountNumber: "4521", BankName: "HDFC", Balance: 100000, IsActive: true},
	}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-1",
			Subject: "Alert : Update for your HDFC Bank A/c XX4521",
			Body:    "Dear Customer,\nRs.450.00 has been debited from account **4521 on 15-06-2026 to VPA swiggy@axisbank (UPI Ref No. 415678901234).\nAvailable Balance: Rs.99,550.00",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipeline(db, "user1")
	p.Process(context.Background(), emails)

	if db.accounts[0].Balance != 99550.00 {
		t.Errorf("expected balance 99550, got %f", db.accounts[0].Balance)
	}
}

func TestPipelineSkipsUnparseable(t *testing.T) {
	db := &mockDB{}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-promo",
			Subject: "Your credit card statement is ready",
			Body:    "Dear Customer, Your monthly statement is ready for download.",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipeline(db, "user1")
	result := p.Process(context.Background(), emails)

	if result.Skipped != 1 {
		t.Errorf("expected 1 skipped, got %d", result.Skipped)
	}
	if len(db.transactions) != 0 {
		t.Error("no transaction should be created for unparseable emails")
	}
}
