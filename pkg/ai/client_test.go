package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCategorizeTransaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-api-key" {
			t.Error("expected auth header")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("expected json content type")
		}

		resp := CompletionResponse{
			Choices: []Choice{
				{Message: Message{Content: `{"category": "Food & Dining", "confidence": 0.92, "reasoning": "Swiggy is a food delivery app"}`}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		Model:   "meta/llama-3.1-70b-instruct",
	})

	result, err := client.CategorizeTransaction(context.Background(), CategorizeRequest{
		Merchant:    "SWIGGY",
		Description: "UPI-swiggy@axisbank",
		Amount:      450,
		Type:        "debit",
		Channel:     "UPI",
		Categories:  []string{"Food & Dining", "Transport", "Shopping", "Entertainment"},
	})

	if err != nil {
		t.Fatalf("CategorizeTransaction failed: %v", err)
	}
	if result.Category != "Food & Dining" {
		t.Errorf("expected Food & Dining, got %q", result.Category)
	}
	if result.Confidence < 0.9 {
		t.Errorf("expected confidence >= 0.9, got %f", result.Confidence)
	}
	if result.Reasoning == "" {
		t.Error("expected reasoning")
	}
}

func TestCategorizeTransactionAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error": "rate limited"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "meta/llama-3.1-70b-instruct",
	})

	_, err := client.CategorizeTransaction(context.Background(), CategorizeRequest{
		Merchant: "TEST",
		Amount:   100,
	})

	if err == nil {
		t.Error("expected error on API failure")
	}
}

func TestCategorizeTransactionMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := CompletionResponse{
			Choices: []Choice{
				{Message: Message{Content: "not valid json at all"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
		Model:   "meta/llama-3.1-70b-instruct",
	})

	_, err := client.CategorizeTransaction(context.Background(), CategorizeRequest{
		Merchant: "TEST",
		Amount:   100,
	})

	if err == nil {
		t.Error("expected error on malformed AI response")
	}
}

func TestCategorizeTransactionNoAPIKey(t *testing.T) {
	client := NewClient(Config{
		APIKey:  "",
		BaseURL: "http://unused",
		Model:   "meta/llama-3.1-70b-instruct",
	})

	_, err := client.CategorizeTransaction(context.Background(), CategorizeRequest{
		Merchant: "TEST",
	})

	if err == nil {
		t.Error("expected error when no API key")
	}
}

func TestBuildPrompt(t *testing.T) {
	prompt := buildPrompt(CategorizeRequest{
		Merchant:    "SWIGGY",
		Description: "UPI-swiggy@axisbank",
		Amount:      450,
		Type:        "debit",
		Channel:     "UPI",
		Categories:  []string{"Food & Dining", "Transport"},
	})

	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if !containsStr(prompt, "SWIGGY") {
		t.Error("prompt should contain merchant")
	}
	if !containsStr(prompt, "Food & Dining") {
		t.Error("prompt should contain categories")
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
