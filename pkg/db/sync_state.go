package db

import (
	"context"
	"fmt"

	"github.com/innacy/finance-agent/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const syncStateCollection = "sync_state"

func (c *Client) GetSyncState(ctx context.Context, userID, source string) (*models.SyncState, error) {
	filter := bson.M{
		"user_id": userID,
		"source":  source,
	}

	var state models.SyncState
	err := c.database.Collection(syncStateCollection).FindOne(ctx, filter).Decode(&state)
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			return nil, nil
		}
		return nil, fmt.Errorf("getting sync state: %w", err)
	}
	return &state, nil
}

func (c *Client) UpsertSyncState(ctx context.Context, state *models.SyncState) error {
	filter := bson.M{
		"user_id": state.UserID,
		"source":  state.Source,
	}

	update := bson.M{
		"$set": bson.M{
			"last_sync_time":  state.LastSyncTime,
			"last_message_id": state.LastMessageID,
			"total_processed": state.TotalProcessed,
			"last_error":      state.LastError,
			"status":          state.Status,
		},
		"$setOnInsert": bson.M{
			"user_id": state.UserID,
			"source":  state.Source,
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	_, err := c.database.Collection(syncStateCollection).UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("upserting sync state: %w", err)
	}
	return nil
}
