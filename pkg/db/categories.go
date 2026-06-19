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
	categoriesCollection    = "categories"
	merchantMemCollection   = "merchant_memory"
	categoryRulesCollection = "category_rules"
)

func (c *Client) CreateCategory(ctx context.Context, cat *models.Category) error {
	if cat.CreatedAt.IsZero() {
		cat.CreatedAt = time.Now()
	}
	result, err := c.database.Collection(categoriesCollection).InsertOne(ctx, cat)
	if err != nil {
		return fmt.Errorf("creating category: %w", err)
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		cat.ID = oid
	}
	return nil
}

func (c *Client) GetCategories(ctx context.Context, userID string) ([]models.Category, error) {
	cursor, err := c.database.Collection(categoriesCollection).Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("finding categories: %w", err)
	}
	defer cursor.Close(ctx)

	var cats []models.Category
	if err := cursor.All(ctx, &cats); err != nil {
		return nil, fmt.Errorf("decoding categories: %w", err)
	}
	return cats, nil
}

func (c *Client) SeedDefaultCategories(ctx context.Context, userID string) error {
	defaults := []models.Category{
		{Name: "Food & Dining", Icon: "🍕", Keywords: []string{"swiggy", "zomato", "restaurant", "cafe", "food"}},
		{Name: "Transport", Icon: "🚗", Keywords: []string{"uber", "ola", "rapido", "metro", "fuel", "petrol"}},
		{Name: "Shopping", Icon: "🛍️", Keywords: []string{"amazon", "flipkart", "myntra", "ajio"}},
		{Name: "Bills & Utilities", Icon: "💡", Keywords: []string{"electricity", "water", "gas", "broadband", "recharge"}},
		{Name: "Entertainment", Icon: "🎬", Keywords: []string{"netflix", "spotify", "hotstar", "prime", "movie"}},
		{Name: "Health", Icon: "🏥", Keywords: []string{"pharmacy", "hospital", "doctor", "apollo", "medplus"}},
		{Name: "Investment", Icon: "📈", Keywords: []string{"mutual fund", "zerodha", "groww", "sip", "nps"}},
		{Name: "Salary", Icon: "💰", Keywords: []string{"salary", "neft", "payroll"}},
		{Name: "Transfer", Icon: "🔄", Keywords: []string{"transfer", "self"}},
		{Name: "ATM", Icon: "🏧", Keywords: []string{"atm", "withdrawal", "cash"}},
		{Name: "EMI", Icon: "🏠", Keywords: []string{"emi", "loan", "bajaj"}},
		{Name: "Subscriptions", Icon: "📱", Keywords: []string{"subscription", "membership", "annual"}},
	}

	for i := range defaults {
		defaults[i].UserID = userID
		defaults[i].IsDefault = true
		defaults[i].CreatedAt = time.Now()
	}

	docs := make([]interface{}, len(defaults))
	for i := range defaults {
		docs[i] = &defaults[i]
	}

	_, err := c.database.Collection(categoriesCollection).InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("seeding categories: %w", err)
	}
	return nil
}

func (c *Client) UpsertMerchantMemory(ctx context.Context, mem *models.MerchantMemory) error {
	if mem.CreatedAt.IsZero() {
		mem.CreatedAt = time.Now()
	}
	mem.LastUsed = time.Now()

	filter := bson.M{
		"user_id":         mem.UserID,
		"normalized_name": mem.NormalizedName,
	}

	update := bson.M{
		"$set": bson.M{
			"merchant_name": mem.MerchantName,
			"category":      mem.Category,
			"sub_category":  mem.SubCategory,
			"confidence":    mem.Confidence,
			"last_used":     mem.LastUsed,
			"source":        mem.Source,
		},
		"$inc": bson.M{
			"times_used": 1,
		},
		"$setOnInsert": bson.M{
			"user_id":         mem.UserID,
			"normalized_name": mem.NormalizedName,
			"created_at":      mem.CreatedAt,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := c.database.Collection(merchantMemCollection).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("upserting merchant memory: %w", err)
	}
	return nil
}

func (c *Client) LookupMerchant(ctx context.Context, userID, normalizedName string) (*models.MerchantMemory, error) {
	filter := bson.M{
		"user_id":         userID,
		"normalized_name": normalizedName,
	}

	var mem models.MerchantMemory
	err := c.database.Collection(merchantMemCollection).FindOne(ctx, filter).Decode(&mem)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, fmt.Errorf("looking up merchant: %w", err)
	}
	return &mem, nil
}

func (c *Client) GetAllMerchantMemory(ctx context.Context, userID string) ([]models.MerchantMemory, error) {
	cursor, err := c.database.Collection(merchantMemCollection).Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("finding merchant memory: %w", err)
	}
	defer cursor.Close(ctx)

	var mems []models.MerchantMemory
	if err := cursor.All(ctx, &mems); err != nil {
		return nil, fmt.Errorf("decoding merchant memory: %w", err)
	}
	return mems, nil
}

func (c *Client) CreateCategoryRule(ctx context.Context, rule *models.CategoryRule) error {
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	result, err := c.database.Collection(categoryRulesCollection).InsertOne(ctx, rule)
	if err != nil {
		return fmt.Errorf("creating category rule: %w", err)
	}
	if oid, ok := result.InsertedID.(bson.ObjectID); ok {
		rule.ID = oid
	}
	return nil
}

func (c *Client) GetCategoryRules(ctx context.Context, userID string) ([]models.CategoryRule, error) {
	opts := options.Find().SetSort(bson.D{{Key: "priority", Value: -1}})
	cursor, err := c.database.Collection(categoryRulesCollection).Find(ctx, bson.M{"user_id": userID, "is_active": true}, opts)
	if err != nil {
		return nil, fmt.Errorf("finding category rules: %w", err)
	}
	defer cursor.Close(ctx)

	var rules []models.CategoryRule
	if err := cursor.All(ctx, &rules); err != nil {
		return nil, fmt.Errorf("decoding rules: %w", err)
	}
	return rules, nil
}
