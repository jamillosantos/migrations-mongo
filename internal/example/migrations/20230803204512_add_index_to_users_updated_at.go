package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ = Migration(func(ctx context.Context, db *mongo.Database) error {
	c := db.Collection("users")
	_, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "updated_at", Value: 1}},
		Options: options.Index().
			SetName("idx_users_updated_at"),
	})
	return err
})
