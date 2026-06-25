package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/config"
	"github.com/innacy/finance-agent/pkg/db"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func setupTestServer(t *testing.T) (*Server, func()) {
	t.Helper()

	dbCfg := &config.DBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "finance-agent-api-test",
		Timeout:  5 * time.Second,
	}
	client, err := db.NewClient(dbCfg)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	if err := client.Ping(context.Background()); err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}

	srv := NewServer(client, "test-user")
	cleanup := func() {
		client.DB().Drop(context.Background())
		client.Disconnect(context.Background())
	}
	return srv, cleanup
}

func TestGetAccountsEmpty(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/accounts", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []models.BankAccount
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 0 {
		t.Errorf("expected empty accounts, got %d", len(resp))
	}
}

func TestGetAccountsWithData(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	srv.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "4521",
		Balance: 100000, Currency: "INR", IsActive: true,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/accounts", nil)
	srv.Router().ServeHTTP(w, req)

	var resp []models.BankAccount
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 account, got %d", len(resp))
	}
	if resp[0].BankName != "HDFC" {
		t.Errorf("expected HDFC, got %q", resp[0].BankName)
	}
}

func TestGetTransactions(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", Type: "debit", Amount: 450,
		Description: "SWIGGY", Category: "Food & Dining",
		TransactionDate: time.Now(), Source: "gmail_alert",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/transactions", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []models.Transaction
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) != 1 {
		t.Errorf("expected 1 transaction, got %d", len(resp))
	}
}

func TestGetOverview(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	srv.db.CreateAccount(ctx, &models.BankAccount{
		UserID: "test-user", BankName: "HDFC", AccountNumber: "4521",
		Balance: 100000, Currency: "INR", IsActive: true,
	})
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", Type: "debit", Amount: 450,
		TransactionDate: time.Now(), Source: "gmail_alert",
	})
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", Type: "credit", Amount: 50000,
		TransactionDate: time.Now(), Source: "gmail_alert",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/overview", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp OverviewResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.TotalBalance != 100000 {
		t.Errorf("expected balance 100000, got %f", resp.TotalBalance)
	}
	if resp.TotalAccounts != 1 {
		t.Errorf("expected 1 account, got %d", resp.TotalAccounts)
	}
}

func TestGetCategories(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	srv.db.SeedDefaultCategories(ctx, "test-user")

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/categories", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []models.Category
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) < 8 {
		t.Errorf("expected at least 8 categories, got %d", len(resp))
	}
}

func TestGetBrainStatus(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	srv.db.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID: "test-user", TotalPredictions: 100,
		CorrectPredictions: 85, Accuracy: 0.85, TrainingSize: 200,
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/brain/status", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp models.BrainMetrics
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Accuracy != 0.85 {
		t.Errorf("expected accuracy 0.85, got %f", resp.Accuracy)
	}
}

func TestGetSpendByCategory(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()
	accID := bson.NewObjectID()
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", AccountID: accID, Type: "debit", Amount: 450,
		Category: "Food & Dining", TransactionDate: time.Now(), Source: "gmail_alert",
	})
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", AccountID: accID, Type: "debit", Amount: 200,
		Category: "Food & Dining", TransactionDate: time.Now(), Source: "gmail_alert",
	})
	srv.db.CreateTransaction(ctx, &models.Transaction{
		UserID: "test-user", AccountID: accID, Type: "debit", Amount: 1000,
		Category: "Transport", TransactionDate: time.Now(), Source: "gmail_alert",
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/spend/categories", nil)
	srv.Router().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var resp []CategorySpend
	json.NewDecoder(w.Body).Decode(&resp)
	if len(resp) < 2 {
		t.Errorf("expected at least 2 categories, got %d", len(resp))
	}
}
