package categorizer

import (
	"context"
	"testing"

	"github.com/innacy/finance-agent/internal/models"
	"github.com/innacy/finance-agent/pkg/ai"
)

type mockAIClient struct {
	result *ai.CategorizeResult
	err    error
	called bool
}

func (m *mockAIClient) CategorizeTransaction(ctx context.Context, req ai.CategorizeRequest) (*ai.CategorizeResult, error) {
	m.called = true
	return m.result, m.err
}

func TestAIFallbackUsedWhenOthersFail(t *testing.T) {
	db := &mockCatDB{}
	aiClient := &mockAIClient{
		result: &ai.CategorizeResult{
			Category:   "Food & Dining",
			Confidence: 0.88,
			Reasoning:  "Swiggy is a food delivery platform",
		},
	}

	c := NewWithAI(db, "user1", 0.8, nil, aiClient, 0.6)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant:    "SWIGGY",
		Description: "UPI-swiggy@axisbank",
		Amount:      450,
		Type:        "debit",
		Channel:     "UPI",
	})

	if !aiClient.called {
		t.Error("AI client should have been called")
	}
	if result.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining from AI, got %q", result.Category)
	}
	if result.Method != "ai" {
		t.Errorf("expected ai method, got %q", result.Method)
	}
	if result.Confidence != 0.88 {
		t.Errorf("expected confidence 0.88, got %f", result.Confidence)
	}
}

func TestAINotCalledWhenHigherLayerSucceeds(t *testing.T) {
	db := &mockCatDB{
		rules: []models.CategoryRule{
			{UserID: "user1", Pattern: "swiggy", Field: "merchant", Category: "Food & Dining", Priority: 10, IsActive: true},
		},
	}
	aiClient := &mockAIClient{
		result: &ai.CategorizeResult{Category: "Transport", Confidence: 0.9},
	}

	c := NewWithAI(db, "user1", 0.8, nil, aiClient, 0.6)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant: "SWIGGY",
	})

	if aiClient.called {
		t.Error("AI should not be called when rule matches")
	}
	if result.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining from rule, got %q", result.Category)
	}
}

func TestAIBelowThresholdReturnsUncategorized(t *testing.T) {
	db := &mockCatDB{}
	aiClient := &mockAIClient{
		result: &ai.CategorizeResult{
			Category:   "Maybe Food",
			Confidence: 0.4,
		},
	}

	c := NewWithAI(db, "user1", 0.8, nil, aiClient, 0.6)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant: "UNKNOWN",
	})

	if result.Category != "Uncategorized" {
		t.Errorf("AI below threshold should return Uncategorized, got %q", result.Category)
	}
}

func TestAIErrorFallsThrough(t *testing.T) {
	db := &mockCatDB{}
	aiClient := &mockAIClient{
		err: ai.ErrNoAPIKey,
	}

	c := NewWithAI(db, "user1", 0.8, nil, aiClient, 0.6)

	result := c.Categorize(context.Background(), &CategorizeInput{
		Merchant: "TEST",
	})

	if result.Category != "Uncategorized" {
		t.Errorf("AI error should fallback to Uncategorized, got %q", result.Category)
	}
}
