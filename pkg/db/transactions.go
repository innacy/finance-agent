package db

import (
	"context"
	"fmt"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const transactionsCollection = "transactions"

func (c *Client) CreateTransaction(ctx context.Context, txn *models.Transaction) error {
	if txn.CreatedAt.IsZero() {
		txn.CreatedAt = time.Now()
	}

	result, err := c.database.Collection(transactionsCollection).InsertOne(ctx, txn)
	if err != nil {
		return fmt.Errorf("inserting transaction: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		txn.ID = oid
	}
	return nil
}

func (c *Client) GetTransactionsByUser(ctx context.Context, userID string, days int, skip, limit int64) ([]models.Transaction, error) {
	since := time.Now().AddDate(0, 0, -days)

	filter := bson.M{
		"user_id":          userID,
		"transaction_date": bson.M{"$gte": since},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "transaction_date", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := c.database.Collection(transactionsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("finding transactions: %w", err)
	}
	defer cursor.Close(ctx)

	var txns []models.Transaction
	if err := cursor.All(ctx, &txns); err != nil {
		return nil, fmt.Errorf("decoding transactions: %w", err)
	}
	return txns, nil
}

func (c *Client) GetPendingReviewTransactions(ctx context.Context, userID string, limit int64) ([]models.Transaction, error) {
	filter := bson.M{
		"user_id":       userID,
		"review_status": "pending_review",
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "transaction_date", Value: -1}}).
		SetLimit(limit)

	cursor, err := c.database.Collection(transactionsCollection).Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("finding pending review: %w", err)
	}
	defer cursor.Close(ctx)

	var txns []models.Transaction
	if err := cursor.All(ctx, &txns); err != nil {
		return nil, fmt.Errorf("decoding pending review: %w", err)
	}
	return txns, nil
}

func (c *Client) TransactionExists(ctx context.Context, userID string, accountID bson.ObjectID, date time.Time, amount float64, reference string) (bool, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"user_id":    userID,
		"account_id": accountID,
		"amount":     amount,
		"reference":  reference,
		"transaction_date": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	count, err := c.database.Collection(transactionsCollection).CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("checking transaction existence: %w", err)
	}
	return count > 0, nil
}
