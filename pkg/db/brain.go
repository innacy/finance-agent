package db

import (
	"context"
	"fmt"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	trainingDataCollection = "training_data"
	brainMetricsCollection = "brain_metrics"
)

func (c *Client) AddTrainingData(ctx context.Context, td *models.TrainingData) error {
	if td.CreatedAt.IsZero() {
		td.CreatedAt = time.Now()
	}
	if td.Weight == 0 {
		td.Weight = 1.0
	}

	result, err := c.database.Collection(trainingDataCollection).InsertOne(ctx, td)
	if err != nil {
		return fmt.Errorf("adding training data: %w", err)
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		td.ID = oid
	}
	return nil
}

func (c *Client) GetTrainingData(ctx context.Context, userID string) ([]models.TrainingData, error) {
	cursor, err := c.database.Collection(trainingDataCollection).Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("finding training data: %w", err)
	}
	defer cursor.Close(ctx)

	var data []models.TrainingData
	if err := cursor.All(ctx, &data); err != nil {
		return nil, fmt.Errorf("decoding training data: %w", err)
	}
	return data, nil
}

func (c *Client) GetTrainingDataCount(ctx context.Context, userID string) (int64, error) {
	count, err := c.database.Collection(trainingDataCollection).CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return 0, fmt.Errorf("counting training data: %w", err)
	}
	return count, nil
}

func (c *Client) UpsertBrainMetrics(ctx context.Context, metrics *models.BrainMetrics) error {
	metrics.UpdatedAt = time.Now()

	filter := bson.M{"user_id": metrics.UserID}
	update := bson.M{
		"$set": bson.M{
			"total_predictions":   metrics.TotalPredictions,
			"correct_predictions": metrics.CorrectPredictions,
			"user_corrections":    metrics.UserCorrections,
			"training_size":       metrics.TrainingSize,
			"accuracy":            metrics.Accuracy,
			"last_trained_at":     metrics.LastTrainedAt,
			"top_categories":      metrics.TopCategories,
			"updated_at":          metrics.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"user_id": metrics.UserID,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := c.database.Collection(brainMetricsCollection).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("upserting brain metrics: %w", err)
	}
	return nil
}

func (c *Client) GetBrainMetrics(ctx context.Context, userID string) (*models.BrainMetrics, error) {
	var metrics models.BrainMetrics
	err := c.database.Collection(brainMetricsCollection).FindOne(ctx, bson.M{"user_id": userID}).Decode(&metrics)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, fmt.Errorf("getting brain metrics: %w", err)
	}
	return &metrics, nil
}

func (c *Client) IncrementPrediction(ctx context.Context, userID string, correct bool) error {
	filter := bson.M{"user_id": userID}
	inc := bson.M{"total_predictions": 1}
	if correct {
		inc["correct_predictions"] = 1
	}

	update := bson.M{
		"$inc": inc,
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := c.database.Collection(brainMetricsCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("incrementing prediction: %w", err)
	}
	return nil
}

func (c *Client) IncrementCorrection(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": bson.M{"user_corrections": 1},
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := c.database.Collection(brainMetricsCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("incrementing correction: %w", err)
	}
	return nil
}
