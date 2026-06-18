package db

import (
	"context"
	"fmt"
	"time"

	"github.com/innacy/finance-agent/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const accountsCollection = "accounts"

func (c *Client) CreateAccount(ctx context.Context, acc *models.BankAccount) error {
	if acc.LastUpdated.IsZero() {
		acc.LastUpdated = time.Now()
	}

	result, err := c.database.Collection(accountsCollection).InsertOne(ctx, acc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("account %s/%s already exists", acc.BankName, acc.AccountNumber)
		}
		return fmt.Errorf("inserting account: %w", err)
	}

	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		acc.ID = oid
	}
	return nil
}

func (c *Client) GetAccountsByUser(ctx context.Context, userID string) ([]models.BankAccount, error) {
	filter := bson.M{
		"user_id":   userID,
		"is_active": true,
	}

	cursor, err := c.database.Collection(accountsCollection).Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("finding accounts: %w", err)
	}
	defer cursor.Close(ctx)

	var accounts []models.BankAccount
	if err := cursor.All(ctx, &accounts); err != nil {
		return nil, fmt.Errorf("decoding accounts: %w", err)
	}
	return accounts, nil
}

func (c *Client) UpdateBalance(ctx context.Context, accountID bson.ObjectID, newBalance float64) error {
	filter := bson.M{"_id": accountID}
	update := bson.M{
		"$set": bson.M{
			"balance":      newBalance,
			"last_updated": time.Now(),
		},
	}

	result, err := c.database.Collection(accountsCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("updating balance: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (c *Client) GetTotalBalance(ctx context.Context, userID string) (float64, error) {
	accounts, err := c.GetAccountsByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	var total float64
	for _, acc := range accounts {
		total += acc.Balance
	}
	return total, nil
}

func (c *Client) DeactivateAccount(ctx context.Context, accountID bson.ObjectID) error {
	filter := bson.M{"_id": accountID}
	update := bson.M{
		"$set": bson.M{
			"is_active":    false,
			"last_updated": time.Now(),
		},
	}

	result, err := c.database.Collection(accountsCollection).UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("deactivating account: %w", err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("account not found")
	}
	return nil
}

func (c *Client) EnsureIndexes(ctx context.Context) error {
	col := c.database.Collection(accountsCollection)

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "bank_name", Value: 1},
			{Key: "account_number", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := col.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("creating account index: %w", err)
	}
	return nil
}
