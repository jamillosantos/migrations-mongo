package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var _ = Migration(func(ctx context.Context, db *mongo.Database) error {
	err := db.CreateCollection(ctx, "users")
	if err != nil {
		return err
	}
	c := DB.Collection("users")
	_, err = c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{"email", 1}},
		Options: options.Index().
			SetName("idx_unique_users_email").
			SetUnique(true),
	})
	return err
})
