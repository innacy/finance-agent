package categorizer

import (
	"context"
	"testing"

	"github.com/innacy/finance-agent/internal/models"
)

type mockCatDB struct {
	merchants []models.MerchantMemory
	rules     []models.CategoryRule
	categories []models.Category
}

func (m *mockCatDB) LookupMerchant(ctx context.Context, userID, normalized string) (*models.MerchantMemory, error) {
	for _, mem := range m.merchants {
		if mem.UserID == userID && mem.NormalizedName == normalized {
			return &mem, nil
		}
	}
	return nil, nil
}

func (m *mockCatDB) GetAllMerchantMemory(ctx context.Context, userID string) ([]models.MerchantMemory, error) {
	var result []models.MerchantMemory
	for _, mem := range m.merchants {
		if mem.UserID == userID {
			result = append(result, mem)
		}
	}
	return result, nil
}

func (m *mockCatDB) UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error {
	for i, existing := range m.merchants {
		if existing.UserID == mem.UserID && existing.NormalizedName == mem.NormalizedName {
			m.merchants[i].Category = mem.Category
			m.merchants[i].TimesUsed++
			m.merchants[i].Confidence = mem.Confidence
			return nil
		}
	}
	m.merchants = append(m.merchants, *mem)
	return nil
}

func (m *mockCatDB) GetCategoryRules(ctx context.Context, userID string) ([]models.CategoryRule, error) {
	var result []models.CategoryRule
	for _, r := range m.rules {
		if r.UserID == userID && r.IsActive {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockCatDB) GetCategories(ctx context.Context, userID string) ([]models.Category, error) {
	var result []models.Category
	for _, c := range m.categories {
		if c.UserID == userID {
			result = append(result, c)
		}
	}
	return result, nil
}

func TestMerchantMemoryExactMatch(t *testing.T) {
	db := &mockCatDB{
		merchants: []models.MerchantMemory{
			{UserID: "user1", NormalizedName: "swiggy", Category: "Food & Dining", Confidence: 0.95, TimesUsed: 10},
		},
	}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "SWIGGY",
		Description: "UPI-swiggy@axisbank",
	})

	if result.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining, got %q", result.Category)
	}
	if result.Method != "merchant_memory" {
		t.Errorf("expected merchant_memory method, got %q", result.Method)
	}
	if result.Confidence < 0.9 {
		t.Errorf("expected high confidence, got %f", result.Confidence)
	}
}

func TestMerchantMemoryFuzzyMatch(t *testing.T) {
	db := &mockCatDB{
		merchants: []models.MerchantMemory{
			{UserID: "user1", NormalizedName: "swiggy", Category: "Food & Dining", Confidence: 0.95, TimesUsed: 5},
		},
	}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "SWIGY",
		Description: "UPI-swigy@axisbank",
	})

	if result.Category != "Food & Dining" {
		t.Errorf("fuzzy match should resolve 'SWIGY' to swiggy, got category %q", result.Category)
	}
	if result.Method != "fuzzy_match" {
		t.Errorf("expected fuzzy_match method, got %q", result.Method)
	}
}

func TestRuleBasedCategorization(t *testing.T) {
	db := &mockCatDB{
		rules: []models.CategoryRule{
			{UserID: "user1", Pattern: "swiggy|zomato|uber.eats", Field: "merchant", Category: "Food & Dining", Priority: 10, IsActive: true},
			{UserID: "user1", Pattern: "uber|ola|rapido", Field: "merchant", Category: "Transport", Priority: 5, IsActive: true},
		},
	}

	c := New(db, "user1", 0.8)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "ZOMATO",
		Description: "UPI-zomato@paytm",
	})
	if result.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining from rule, got %q", result.Category)
	}
	if result.Method != "rule" {
		t.Errorf("expected rule method, got %q", result.Method)
	}
}

func TestRuleMatchesDescription(t *testing.T) {
	db := &mockCatDB{
		rules: []models.CategoryRule{
			{UserID: "user1", Pattern: "netflix|hotstar|prime", Field: "description", Category: "Entertainment", Priority: 10, IsActive: true},
		},
	}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "UNKNOWN",
		Description: "Netflix subscription renewal",
	})
	if result.Category != "Entertainment" {
		t.Errorf("expected Entertainment from description rule, got %q", result.Category)
	}
}

func TestKeywordCategorization(t *testing.T) {
	db := &mockCatDB{
		categories: []models.Category{
			{UserID: "user1", Name: "Food & Dining", Keywords: []string{"swiggy", "zomato", "restaurant"}},
			{UserID: "user1", Name: "Transport", Keywords: []string{"uber", "ola", "metro"}},
		},
	}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "METRO STATION",
		Description: "metro recharge",
	})
	if result.Category != "Transport" {
		t.Errorf("expected Transport from keyword, got %q", result.Category)
	}
	if result.Method != "keyword" {
		t.Errorf("expected keyword method, got %q", result.Method)
	}
}

func TestPatternDetectionATM(t *testing.T) {
	db := &mockCatDB{}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Channel:     "ATM",
		Description: "ATM Withdrawal",
	})
	if result.Category != "ATM" {
		t.Errorf("expected ATM category, got %q", result.Category)
	}
	if result.Method != "pattern" {
		t.Errorf("expected pattern method, got %q", result.Method)
	}
}

func TestPatternDetectionSalary(t *testing.T) {
	db := &mockCatDB{}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Type:        "credit",
		Channel:     "NEFT",
		Amount:      85000,
		Description: "NEFT-ACME CORP LTD",
	})
	if result.Category != "Salary" {
		t.Errorf("large NEFT credit should be categorized as Salary, got %q", result.Category)
	}
}

func TestUncategorizedFallback(t *testing.T) {
	db := &mockCatDB{}

	c := New(db, "user1", 0.8)
	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "RANDOM_VENDOR_123",
		Description: "some unknown txn",
	})
	if result.Category != "Uncategorized" {
		t.Errorf("expected Uncategorized fallback, got %q", result.Category)
	}
	if result.NeedsReview {
		t.Error("uncategorized below threshold should not need review by default")
	}
}

func TestLearnFromCorrection(t *testing.T) {
	db := &mockCatDB{}

	c := New(db, "user1", 0.8)
	err := c.Learn(context.Background(), "DOMINOS PIZZA", "Food & Dining", "user_correction")
	if err != nil {
		t.Fatal(err)
	}

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant: "DOMINOS PIZZA",
	})
	if result.Category != "Food & Dining" {
		t.Errorf("after learning, should categorize DOMINOS PIZZA as Food & Dining, got %q", result.Category)
	}
}
