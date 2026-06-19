package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Category struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string        `bson:"user_id" json:"user_id"`
	Name        string        `bson:"name" json:"name"`
	Icon        string        `bson:"icon" json:"icon"`
	Color       string        `bson:"color" json:"color"`
	IsDefault   bool          `bson:"is_default" json:"is_default"`
	Keywords    []string      `bson:"keywords,omitempty" json:"keywords,omitempty"`
	SubCategory []string      `bson:"sub_categories,omitempty" json:"sub_categories,omitempty"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
}

type MerchantMemory struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       string        `bson:"user_id" json:"user_id"`
	MerchantName string        `bson:"merchant_name" json:"merchant_name"`
	NormalizedName string      `bson:"normalized_name" json:"normalized_name"`
	Category     string        `bson:"category" json:"category"`
	SubCategory  string        `bson:"sub_category" json:"sub_category"`
	Confidence   float64       `bson:"confidence" json:"confidence"`
	TimesUsed    int           `bson:"times_used" json:"times_used"`
	LastUsed     time.Time     `bson:"last_used" json:"last_used"`
	Source       string        `bson:"source" json:"source"`
	CreatedAt    time.Time     `bson:"created_at" json:"created_at"`
}

type CategoryRule struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      string        `bson:"user_id" json:"user_id"`
	Pattern     string        `bson:"pattern" json:"pattern"`
	Field       string        `bson:"field" json:"field"`
	Category    string        `bson:"category" json:"category"`
	SubCategory string        `bson:"sub_category" json:"sub_category"`
	Priority    int           `bson:"priority" json:"priority"`
	IsActive    bool          `bson:"is_active" json:"is_active"`
	CreatedAt   time.Time     `bson:"created_at" json:"created_at"`
}
