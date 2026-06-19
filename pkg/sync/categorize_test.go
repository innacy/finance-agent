package sync

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/gmail"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type catMockDB struct {
	mockDB
	merchants  []models.MerchantMemory
	rules      []models.CategoryRule
	categories []models.Category
}

func (m *catMockDB) LookupMerchant(ctx context.Context, userID, normalized string) (*models.MerchantMemory, error) {
	for _, mem := range m.merchants {
		if mem.UserID == userID && mem.NormalizedName == normalized {
			return &mem, nil
		}
	}
	return nil, nil
}

func (m *catMockDB) GetAllMerchantMemory(ctx context.Context, userID string) ([]models.MerchantMemory, error) {
	var result []models.MerchantMemory
	for _, mem := range m.merchants {
		if mem.UserID == userID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *catMockDB) UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error {
	m.merchants = append(m.merchants, *mem)
	return nil
}

func (m *catMockDB) GetCategoryRules(ctx context.Context, userID string) ([]models.CategoryRule, error) {
	var result []models.CategoryRule
	for _, r := range m.rules {
		if r.UserID == userID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *catMockDB) GetCategories(ctx context.Context, userID string) ([]models.Category, error) {
	var result []models.Category
	for _, c := range m.categories {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

func TestPipelineCategorizesViaRule(t *testing.T) {
	db := &catMockDB{}
	db.accounts = []models.BankAccount{
		{ID: bson.NewObjectID(), UserID: "user1", AccountNumber: "4521", BankName: "HDFC", IsActive: true},
	}
	db.rules = []models.CategoryRule{
		{UserID: "user1", Pattern: "swiggy|zomato", Field: "merchant", Category: "Food & Dining", Priority: 10, IsActive: true},
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

	p := NewPipelineWithCategorizer(db, "user1", 0.8)
	p.Process(context.Background(), emails)

	if len(db.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(db.transactions))
	}
	txn := db.transactions[0]
	if txn.Category != "Food & Dining" {
		t.Errorf("expected category 'Food & Dining', got %q", txn.Category)
	}
	if txn.CategorizedBy != "rule" {
		t.Errorf("expected categorized_by 'rule', got %q", txn.CategorizedBy)
	}
}

func TestPipelineCategorizesATMByPattern(t *testing.T) {
	db := &catMockDB{}
	db.accounts = []models.BankAccount{
		{ID: bson.NewObjectID(), UserID: "user1", AccountNumber: "4521", BankName: "HDFC", IsActive: true},
	}

	emails := []gmail.RawMessage{
		{
			ID:      "msg-atm",
			Subject: "Alert : Rs.10,000.00 withdrawn from ATM for your HDFC Bank A/c",
			Body:    "Dear Customer,\nRs.10,000.00 has been withdrawn from your A/c **4521 at ATM on 10-06-2026 (Ref No. ATM-98765).\nAvailable Balance: Rs.89,550.00",
			From:    "alerts@hdfcbank.net",
			Date:    time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC),
		},
	}

	p := NewPipelineWithCategorizer(db, "user1", 0.8)
	p.Process(context.Background(), emails)

	if len(db.transactions) != 1 {
		t.Fatalf("expected 1 transaction, got %d", len(db.transactions))
	}
	if db.transactions[0].Category != "ATM" {
		t.Errorf("expected ATM category, got %q", db.transactions[0].Category)
	}
}

func TestPipelineLearnsFromNewTransaction(t *testing.T) {
	db := &catMockDB{}
	db.accounts = []models.BankAccount{
		{ID: bson.NewObjectID(), UserID: "user1", AccountNumber: "4521", BankName: "HDFC", IsActive: true},
	}
	db.rules = []models.CategoryRule{
		{UserID: "user1", Pattern: "swiggy", Field: "merchant", Category: "Food & Dining", Priority: 10, IsActive: true},
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

	p := NewPipelineWithCategorizer(db, "user1", 0.8)
	p.Process(context.Background(), emails)

	found := false
	for _, m := range db.merchants {
		if m.NormalizedName == "swiggyaxisbank" || m.Category == "Food & Dining" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected pipeline to learn merchant from successful categorization")
	}
}
