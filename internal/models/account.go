package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type BankAccount struct {
	ID             bson.ObjectID     `bson:"_id,omitempty" json:"id"`
	UserID         string            `bson:"user_id" json:"user_id"`
	BankName       string            `bson:"bank_name" json:"bank_name"`
	AccountNumber  string            `bson:"account_number" json:"account_number"`
	AccountType    string            `bson:"account_type" json:"account_type"`
	Balance        float64           `bson:"balance" json:"balance"`
	Currency       string            `bson:"currency" json:"currency"`
	LastUpdated    time.Time         `bson:"last_updated" json:"last_updated"`
	IsActive       bool              `bson:"is_active" json:"is_active"`
	IsConfirmed    bool              `bson:"is_confirmed" json:"is_confirmed"`
	DetectedFrom   string            `bson:"detected_from" json:"detected_from"`
	Metadata       map[string]string `bson:"metadata,omitempty" json:"metadata,omitempty"`
}
