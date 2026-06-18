package db

import (
	"context"
	"fmt"

	"github.com/innacy/finance-agent/pkg/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Client struct {
	mongo    *mongo.Client
	database *mongo.Database
	dbName   string
}

func NewClient(cfg *config.DBConfig) (*Client, error) {
	if cfg.URI == "" {
		return nil, fmt.Errorf("database URI is required")
	}

	opts := options.Client().ApplyURI(cfg.URI)
	if cfg.Timeout > 0 {
		opts.SetConnectTimeout(cfg.Timeout)
	}

	mongoClient, err := mongo.Connect(opts)
	if err != nil {
		return nil, fmt.Errorf("connecting to mongodb: %w", err)
	}

	return &Client{
		mongo:    mongoClient,
		database: mongoClient.Database(cfg.Database),
		dbName:   cfg.Database,
	}, nil
}

func (c *Client) DatabaseName() string {
	return c.dbName
}

func (c *Client) Ping(ctx context.Context) error {
	return c.mongo.Ping(ctx, nil)
}

func (c *Client) Disconnect(ctx context.Context) error {
	return c.mongo.Disconnect(ctx)
}

func (c *Client) DB() *mongo.Database {
	return c.database
}
