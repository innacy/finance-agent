package db

import (
	"context"
	"testing"

	"github.com/innacy/finance-agent/internal/models"
)

func TestAddTrainingData(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	td := &models.TrainingData{
		UserID:   "user1",
		Text:     "swiggy food delivery upi",
		Category: "Food & Dining",
		Source:   "user_correction",
		Weight:   10.0,
	}

	err := client.AddTrainingData(context.Background(), td)
	if err != nil {
		t.Fatalf("AddTrainingData failed: %v", err)
	}
	if td.ID.IsZero() {
		t.Error("expected ID to be set")
	}
	if td.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestGetTrainingData(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user1", Text: "swiggy food", Category: "Food & Dining", Weight: 1.0,
	})
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user1", Text: "uber ride", Category: "Transport", Weight: 1.0,
	})
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user2", Text: "other user data", Category: "Other", Weight: 1.0,
	})

	data, err := client.GetTrainingData(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Errorf("expected 2 training records for user1, got %d", len(data))
	}
}

func TestGetTrainingDataCount(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user1", Text: "food1", Category: "Food", Weight: 1.0,
	})
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user1", Text: "food2", Category: "Food", Weight: 1.0,
	})
	client.AddTrainingData(ctx, &models.TrainingData{
		UserID: "user1", Text: "transport1", Category: "Transport", Weight: 1.0,
	})

	count, err := client.GetTrainingDataCount(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}
}

func TestUpsertBrainMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	err := client.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID:             "user1",
		TotalPredictions:   100,
		CorrectPredictions: 85,
		UserCorrections:    15,
		TrainingSize:       200,
		Accuracy:           0.85,
	})
	if err != nil {
		t.Fatalf("UpsertBrainMetrics failed: %v", err)
	}

	metrics, err := client.GetBrainMetrics(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}
	if metrics == nil {
		t.Fatal("expected metrics")
	}
	if metrics.Accuracy != 0.85 {
		t.Errorf("expected accuracy 0.85, got %f", metrics.Accuracy)
	}
	if metrics.TotalPredictions != 100 {
		t.Errorf("expected 100 predictions, got %d", metrics.TotalPredictions)
	}
}

func TestIncrementBrainMetrics(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID:             "user1",
		TotalPredictions:   10,
		CorrectPredictions: 8,
		Accuracy:           0.8,
	})

	err := client.IncrementPrediction(ctx, "user1", true)
	if err != nil {
		t.Fatal(err)
	}

	metrics, _ := client.GetBrainMetrics(ctx, "user1")
	if metrics.TotalPredictions != 11 {
		t.Errorf("expected 11 total, got %d", metrics.TotalPredictions)
	}
	if metrics.CorrectPredictions != 9 {
		t.Errorf("expected 9 correct, got %d", metrics.CorrectPredictions)
	}
}

func TestIncrementCorrection(t *testing.T) {
	client, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	client.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID:          "user1",
		UserCorrections: 5,
	})

	err := client.IncrementCorrection(ctx, "user1")
	if err != nil {
		t.Fatal(err)
	}

	metrics, _ := client.GetBrainMetrics(ctx, "user1")
	if metrics.UserCorrections != 6 {
		t.Errorf("expected 6 corrections, got %d", metrics.UserCorrections)
	}
}
