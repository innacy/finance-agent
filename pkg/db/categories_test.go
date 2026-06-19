package db

import (
	"context"
	"testing"

	"github.com/innacy/finance-agent/internal/models"
)

func TestCreateCategory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	cat := &models.Category{
		UserID:    "user1",
		Name:      "Food & Dining",
		Icon:      "🍕",
		Color:     "#FF6B6B",
		IsDefault: true,
		Keywords:  []string{"swiggy", "zomato", "restaurant"},
	}

	err := client.CreateCategory(context.Background(), cat)
	if err != nil {
		t.Fatalf("CreateCategory failed: %v", err)
	}
	if cat.ID.IsZero() {
		t.Error("expected ID to be set")
	}
}

func TestGetCategories(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.CreateCategory(ctx, &models.Category{UserID: "user1", Name: "Food", IsDefault: true})
	client.CreateCategory(ctx, &models.Category{UserID: "user1", Name: "Transport", IsDefault: true})
	client.CreateCategory(ctx, &models.Category{UserID: "user2", Name: "Other", IsDefault: true})

	cats, err := client.GetCategories(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cats) != 2 {
		t.Errorf("expected 2 categories for user1, got %d", len(cats))
	}
}

func TestSeedDefaultCategories(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := client.SeedDefaultCategories(ctx, "user1")
	if err != nil {
		t.Fatalf("SeedDefaultCategories failed: %v", err)
	}

	cats, _ := client.GetCategories(ctx, "user1")
	if len(cats) == 0 {
		t.Error("expected seeded categories")
	}
	if len(cats) < 8 {
		t.Errorf("expected at least 8 default categories, got %d", len(cats))
	}
}

func TestCreateMerchantMemory(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mem := &models.MerchantMemory{
		UserID:         "user1",
		MerchantName:   "SWIGGY",
		NormalizedName: "swiggy",
		Category:       "Food & Dining",
		Confidence:     0.95,
		TimesUsed:      1,
		Source:         "user_correction",
	}

	err := client.UpsertMerchantMemory(context.Background(), mem)
	if err != nil {
		t.Fatalf("UpsertMerchantMemory failed: %v", err)
	}
}

func TestLookupMerchant(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.UpsertMerchantMemory(ctx, &models.MerchantMemory{
		UserID: "user1", MerchantName: "SWIGGY", NormalizedName: "swiggy",
		Category: "Food & Dining", Confidence: 0.95, TimesUsed: 5, Source: "user",
	})

	mem, err := client.LookupMerchant(ctx, "user1", "swiggy")
	if err != nil {
		t.Fatal(err)
	}
	if mem == nil {
		t.Fatal("expected merchant memory")
	}
	if mem.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining, got %q", mem.Category)
	}
}

func TestLookupMerchantNotFound(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	mem, err := client.LookupMerchant(context.Background(), "user1", "unknown-merchant")
	if err != nil {
		t.Fatal(err)
	}
	if mem != nil {
		t.Error("expected nil for unknown merchant")
	}
}

func TestUpsertMerchantIncrementsCount(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.UpsertMerchantMemory(ctx, &models.MerchantMemory{
		UserID: "user1", MerchantName: "SWIGGY", NormalizedName: "swiggy",
		Category: "Food & Dining", Confidence: 0.9, TimesUsed: 1, Source: "rule",
	})
	client.UpsertMerchantMemory(ctx, &models.MerchantMemory{
		UserID: "user1", MerchantName: "SWIGGY", NormalizedName: "swiggy",
		Category: "Food & Dining", Confidence: 0.95, TimesUsed: 1, Source: "user",
	})

	mem, _ := client.LookupMerchant(ctx, "user1", "swiggy")
	if mem.TimesUsed != 2 {
		t.Errorf("expected times_used 2 after upsert, got %d", mem.TimesUsed)
	}
	if mem.Confidence != 0.95 {
		t.Errorf("expected confidence to update to 0.95, got %f", mem.Confidence)
	}
}

func TestCreateCategoryRule(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	rule := &models.CategoryRule{
		UserID:   "user1",
		Pattern:  "swiggy|zomato|uber.eats",
		Field:    "merchant",
		Category: "Food & Dining",
		Priority: 10,
		IsActive: true,
	}

	err := client.CreateCategoryRule(context.Background(), rule)
	if err != nil {
		t.Fatalf("CreateCategoryRule failed: %v", err)
	}
	if rule.ID.IsZero() {
		t.Error("expected ID to be set")
	}
}

func TestGetCategoryRules(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.CreateCategoryRule(ctx, &models.CategoryRule{
		UserID: "user1", Pattern: "swiggy", Field: "merchant",
		Category: "Food", Priority: 10, IsActive: true,
	})
	client.CreateCategoryRule(ctx, &models.CategoryRule{
		UserID: "user1", Pattern: "uber|ola", Field: "merchant",
		Category: "Transport", Priority: 5, IsActive: true,
	})

	rules, err := client.GetCategoryRules(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Priority < rules[1].Priority {
		t.Error("rules should be ordered by priority descending")
	}
}
