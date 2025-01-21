package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ = Migration(func(ctx context.Context, db *mongo.Database) error {
	err := db.CreateCollection(ctx, "users")
	if err != nil {
		return err
	}
	c := DB.Collection("users")
	_, err = c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]int{"email": 1},
		Options: (&options.IndexOptions{}).
			SetName("idx_unique_users_email").
			SetUnique(true),
	})
	return err
})
