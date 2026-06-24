package brain

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/innacy/finance-agent/internal/models"
)

type TrainerDB interface {
	AddTrainingData(ctx context.Context, td *models.TrainingData) error
	GetTrainingData(ctx context.Context, userID string) ([]models.TrainingData, error)
	GetTrainingDataCount(ctx context.Context, userID string) (int64, error)
	UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error
	UpsertBrainMetrics(ctx context.Context, metrics *models.BrainMetrics) error
	IncrementPrediction(ctx context.Context, userID string, correct bool) error
	IncrementCorrection(ctx context.Context, userID string) error
}

type FeedbackInput struct {
	Text              string
	Merchant          string
	PredictedCategory string
	ActualCategory    string
	IsCorrection      bool
}

type Trainer struct {
	db         TrainerDB
	classifier *Classifier
	userID     string
}

func NewTrainer(db TrainerDB, classifier *Classifier, userID string) *Trainer {
	return &Trainer{db: db, classifier: classifier, userID: userID}
}

func (t *Trainer) RecordFeedback(ctx context.Context, input FeedbackInput) error {
	weight := 1.0
	if input.IsCorrection {
		weight = 10.0
	}

	td := &models.TrainingData{
		UserID:       t.userID,
		Text:         input.Text,
		Category:     input.ActualCategory,
		Source:       feedbackSource(input.IsCorrection),
		Weight:       weight,
		IsCorrection: input.IsCorrection,
	}

	if err := t.db.AddTrainingData(ctx, td); err != nil {
		return err
	}

	t.classifier.Train(input.Text, input.ActualCategory, weight)

	if input.IsCorrection {
		t.db.IncrementCorrection(ctx, t.userID)
		t.db.IncrementPrediction(ctx, t.userID, false)

		if input.Merchant != "" {
			normalized := normalizeMerchant(input.Merchant)
			t.db.UpsertMerchantMemory(ctx, &models.MerchantMemory{
				UserID:         t.userID,
				MerchantName:   input.Merchant,
				NormalizedName: normalized,
				Category:       input.ActualCategory,
				Confidence:     1.0,
				TimesUsed:      1,
				Source:         "user_correction",
				LastUsed:       time.Now(),
			})
		}
	} else {
		t.db.IncrementPrediction(ctx, t.userID, true)
	}

	return nil
}

func (t *Trainer) Retrain(ctx context.Context) error {
	data, err := t.db.GetTrainingData(ctx, t.userID)
	if err != nil {
		return err
	}

	newClassifier := NewClassifier()
	for _, td := range data {
		newClassifier.Train(td.Text, td.Category, td.Weight)
	}

	*t.classifier = *newClassifier

	count, _ := t.db.GetTrainingDataCount(ctx, t.userID)
	return t.db.UpsertBrainMetrics(ctx, &models.BrainMetrics{
		UserID:        t.userID,
		TrainingSize:  count,
		LastTrainedAt: time.Now(),
	})
}

func feedbackSource(isCorrection bool) string {
	if isCorrection {
		return "user_correction"
	}
	return "user_confirmed"
}

var reNormalize = regexp.MustCompile(`[^a-z0-9]+`)

func normalizeMerchant(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return reNormalize.ReplaceAllString(s, "")
}
