package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Transaction struct {
	ID              bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID          string        `bson:"user_id" json:"user_id"`
	AccountID       bson.ObjectID `bson:"account_id" json:"account_id"`
	Type            string        `bson:"type" json:"type"`
	Amount          float64       `bson:"amount" json:"amount"`
	BalanceAfter    float64       `bson:"balance_after" json:"balance_after"`
	Description     string        `bson:"description" json:"description"`
	Merchant        string        `bson:"merchant" json:"merchant"`
	Category        string        `bson:"category" json:"category"`
	SubCategory     string        `bson:"sub_category" json:"sub_category"`
	Tags            []string      `bson:"tags,omitempty" json:"tags,omitempty"`
	TransactionDate time.Time     `bson:"transaction_date" json:"transaction_date"`
	ValueDate       time.Time     `bson:"value_date" json:"value_date"`
	Reference       string        `bson:"reference" json:"reference"`
	Channel         string        `bson:"channel" json:"channel"`
	CounterpartyUPI string        `bson:"counterparty_upi" json:"counterparty_upi"`
	Source          string        `bson:"source" json:"source"`
	SourceRef       string        `bson:"source_ref" json:"source_ref"`
	CategorizedBy   string        `bson:"categorized_by" json:"categorized_by"`
	Confidence      float64       `bson:"confidence" json:"confidence"`
	ReviewStatus    string        `bson:"review_status" json:"review_status"`
	IsRecurring     bool          `bson:"is_recurring" json:"is_recurring"`
	CreatedAt       time.Time     `bson:"created_at" json:"created_at"`
}

type SyncState struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID         string        `bson:"user_id" json:"user_id"`
	Source         string        `bson:"source" json:"source"`
	LastSyncTime   time.Time     `bson:"last_sync_time" json:"last_sync_time"`
	LastMessageID  string        `bson:"last_message_id" json:"last_message_id"`
	TotalProcessed int64         `bson:"total_processed" json:"total_processed"`
	LastError      string        `bson:"last_error" json:"last_error"`
	Status         string        `bson:"status" json:"status"`
}
