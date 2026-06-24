package brain

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/internal/models"
)

type mockTrainerDB struct {
	training   []models.TrainingData
	merchants  []models.MerchantMemory
	metrics    *models.BrainMetrics
}

func (m *mockTrainerDB) AddTrainingData(ctx context.Context, td *models.TrainingData) error {
	td.CreatedAt = time.Now()
	m.training = append(m.training, *td)
	return nil
}

func (m *mockTrainerDB) GetTrainingData(ctx context.Context, userID string) ([]models.TrainingData, error) {
	var result []models.TrainingData
	for _, td := range m.training {
		if td.UserID == userID {
			result = append(result, td)
		}
	}
	return result, nil
}

func (m *mockTrainerDB) GetTrainingDataCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	for _, td := range m.training {
		if td.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (m *mockTrainerDB) UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error {
	m.merchants = append(m.merchants, *mem)
	return nil
}

func (m *mockTrainerDB) UpsertBrainMetrics(ctx context.Context, metrics *models.BrainMetrics) error {
	m.metrics = metrics
	return nil
}

func (m *mockTrainerDB) IncrementPrediction(ctx context.Context, userID string, correct bool) error {
	if m.metrics == nil {
		m.metrics = &models.BrainMetrics{UserID: userID}
	}
	m.metrics.TotalPredictions++
	if correct {
		m.metrics.CorrectPredictions++
	}
	return nil
}

func (m *mockTrainerDB) IncrementCorrection(ctx context.Context, userID string) error {
	if m.metrics == nil {
		m.metrics = &models.BrainMetrics{UserID: userID}
	}
	m.metrics.UserCorrections++
	return nil
}

func TestTrainerAcceptPrediction(t *testing.T) {
	db := &mockTrainerDB{}
	classifier := NewClassifier()
	trainer := NewTrainer(db, classifier, "user1")

	err := trainer.RecordFeedback(context.Background(), FeedbackInput{
		Text:              "swiggy food delivery",
		PredictedCategory: "Food & Dining",
		ActualCategory:    "Food & Dining",
		IsCorrection:      false,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(db.training) != 1 {
		t.Fatalf("expected 1 training record, got %d", len(db.training))
	}
	if db.training[0].Weight != 1.0 {
		t.Errorf("accepted prediction should have weight 1.0, got %f", db.training[0].Weight)
	}
	if db.metrics.CorrectPredictions != 1 {
		t.Errorf("expected 1 correct prediction, got %d", db.metrics.CorrectPredictions)
	}
}

func TestTrainerCorrection(t *testing.T) {
	db := &mockTrainerDB{}
	classifier := NewClassifier()
	trainer := NewTrainer(db, classifier, "user1")

	err := trainer.RecordFeedback(context.Background(), FeedbackInput{
		Text:              "uber ride to airport",
		PredictedCategory: "Food & Dining",
		ActualCategory:    "Transport",
		IsCorrection:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(db.training) != 1 {
		t.Fatalf("expected 1 training record, got %d", len(db.training))
	}
	if db.training[0].Weight != 10.0 {
		t.Errorf("correction should have weight 10.0, got %f", db.training[0].Weight)
	}
	if db.training[0].Category != "Transport" {
		t.Errorf("should store actual category, got %q", db.training[0].Category)
	}
	if db.training[0].IsCorrection != true {
		t.Error("should be marked as correction")
	}
	if db.metrics.UserCorrections != 1 {
		t.Errorf("expected 1 correction in metrics, got %d", db.metrics.UserCorrections)
	}
}

func TestTrainerRetrainFromDB(t *testing.T) {
	db := &mockTrainerDB{
		training: []models.TrainingData{
			{UserID: "user1", Text: "swiggy food", Category: "Food & Dining", Weight: 1.0},
			{UserID: "user1", Text: "zomato dinner", Category: "Food & Dining", Weight: 1.0},
			{UserID: "user1", Text: "uber ride", Category: "Transport", Weight: 1.0},
			{UserID: "user1", Text: "ola cab", Category: "Transport", Weight: 1.0},
			{UserID: "user1", Text: "amazon shop", Category: "Shopping", Weight: 1.0},
		},
	}
	classifier := NewClassifier()
	trainer := NewTrainer(db, classifier, "user1")

	err := trainer.Retrain(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	stats := classifier.Stats()
	if stats.TotalDocuments != 5 {
		t.Errorf("expected 5 docs after retrain, got %d", stats.TotalDocuments)
	}
	if stats.Categories != 3 {
		t.Errorf("expected 3 categories, got %d", stats.Categories)
	}

	if db.metrics == nil {
		t.Fatal("expected metrics to be updated")
	}
	if db.metrics.TrainingSize != 5 {
		t.Errorf("expected training_size 5, got %d", db.metrics.TrainingSize)
	}
}

func TestTrainerCorrectionUpdatesMerchant(t *testing.T) {
	db := &mockTrainerDB{}
	classifier := NewClassifier()
	trainer := NewTrainer(db, classifier, "user1")

	trainer.RecordFeedback(context.Background(), FeedbackInput{
		Text:              "dominos pizza",
		Merchant:          "DOMINOS PIZZA",
		PredictedCategory: "Shopping",
		ActualCategory:    "Food & Dining",
		IsCorrection:      true,
	})

	if len(db.merchants) != 1 {
		t.Fatalf("expected merchant memory to be updated, got %d", len(db.merchants))
	}
	if db.merchants[0].Category != "Food & Dining" {
		t.Errorf("merchant should be updated to Food & Dining, got %q", db.merchants[0].Category)
	}
}
