package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type TrainingData struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string        `bson:"user_id" json:"user_id"`
	Text        string        `bson:"text" json:"text"`
	Category    string        `bson:"category" json:"category"`
	Source      string        `bson:"source" json:"source"`
	Weight      float64       `bson:"weight" json:"weight"`
	IsCorrection bool         `bson:"is_correction" json:"is_correction"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
}

type BrainMetrics struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          string        `bson:"user_id" json:"user_id"`
	TotalPredictions int64        `bson:"total_predictions" json:"total_predictions"`
	CorrectPredictions int64      `bson:"correct_predictions" json:"correct_predictions"`
	UserCorrections int64         `bson:"user_corrections" json:"user_corrections"`
	TrainingSize    int64         `bson:"training_size" json:"training_size"`
	LastTrainedAt   time.Time     `bson:"last_trained_at" json:"last_trained_at"`
	Accuracy        float64       `bson:"accuracy" json:"accuracy"`
	TopCategories   map[string]int64 `bson:"top_categories" json:"top_categories"`
	UpdatedAt       time.Time     `bson:"updated_at" json:"updated_at"`
}
