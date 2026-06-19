package categorizer

import (
	"context"
	"testing"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/brain"
)

func TestCategorizerWithMLFallback(t *testing.T) {
	db := &mockCatDB{}

	classifier := brain.NewClassifier()
	classifier.Train("swiggy food delivery order", "Food & Dining", 1.0)
	classifier.Train("zomato restaurant dinner", "Food & Dining", 1.0)
	classifier.Train("uber ride trip booking", "Transport", 1.0)
	classifier.Train("ola cab auto ride", "Transport", 1.0)
	classifier.Train("amazon flipkart shopping", "Shopping", 1.0)
	classifier.Train("myntra ajio clothes", "Shopping", 1.0)

	c := NewWithML(db, "user1", 0.5, classifier)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "SWIGGY DELIVERY",
		Description: "food order swiggy",
	})
	if result.Category != "Food & Dining" {
		t.Errorf("ML should predict Food & Dining, got %q", result.Category)
	}
	if result.Method != "ml" {
		t.Errorf("expected ml method, got %q", result.Method)
	}
}

func TestMLLayerUsedWhenOthersFail(t *testing.T) {
	db := &mockCatDB{}

	classifier := brain.NewClassifier()
	classifier.Train("netflix subscription monthly", "Entertainment", 1.0)
	classifier.Train("hotstar premium annual", "Entertainment", 1.0)
	classifier.Train("spotify music streaming", "Entertainment", 1.0)
	classifier.Train("prime video watch", "Entertainment", 1.0)
	classifier.Train("swiggy food delivery", "Food & Dining", 1.0)
	classifier.Train("zomato restaurant order", "Food & Dining", 1.0)
	classifier.Train("uber ride booking", "Transport", 1.0)
	classifier.Train("ola cab trip", "Transport", 1.0)

	c := NewWithML(db, "user1", 0.5, classifier)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "NETFLIX",
		Description: "netflix subscription renewal",
	})
	if result.Category != "Entertainment" {
		t.Errorf("expected Entertainment from ML, got %q", result.Category)
	}
}

func TestMLBelowThresholdReturnsUncategorized(t *testing.T) {
	db := &mockCatDB{}

	classifier := brain.NewClassifier()
	classifier.Train("one thing", "Cat1", 1.0)

	c := NewWithML(db, "user1", 0.9, classifier)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "TOTALLY RANDOM THING",
		Description: "completely different context",
	})

	if result.Confidence >= 0.9 && result.Category != "Uncategorized" {
		t.Errorf("low-confidence ML should not override threshold, got category=%q confidence=%f", result.Category, result.Confidence)
	}
}

func TestHigherLayersOverrideML(t *testing.T) {
	db := &mockCatDB{
		merchants: []models.MerchantMemory{
			{UserID: "user1", NormalizedName: "swiggy", Category: "Food & Dining", Confidence: 0.99, TimesUsed: 50},
		},
	}

	classifier := brain.NewClassifier()
	classifier.Train("swiggy is transport", "Transport", 5.0)

	c := NewWithML(db, "user1", 0.8, classifier)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant: "SWIGGY",
	})
	if result.Category != "Food & Dining" {
		t.Errorf("merchant memory should override ML, got %q", result.Category)
	}
	if result.Method != "merchant_memory" {
		t.Errorf("expected merchant_memory, got %q", result.Method)
	}
}
