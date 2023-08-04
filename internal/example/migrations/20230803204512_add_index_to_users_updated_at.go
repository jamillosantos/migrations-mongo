package migrations

import (
	"context"

	. "github.com/jamillosantos/migrations-fnc"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ = Migration(func(ctx context.Context) error {
	c := DB.Collection("users")
	_, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]int{"updated_at": 1},
		Options: (&options.IndexOptions{}).
			SetName("idx_users_updated_at"),
	})
	return err
})
