package migrations

import (
	"context"

	"github.com/jamillosantos/migrations/v2"
	"github.com/jamillosantos/migrations/v2/fnc"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// Source where the migrations will be loaded added to.
	Source = migrations.NewMemorySource()

	// DB is the global database connection that will be used by the migrations.
	DB *mongo.Database
)

// Migration is a helper function that will wrap the migration function adding the mongo.Database reference.
func Migration(migrationFunc func(ctx context.Context, db *mongo.Database) error) migrations.Migration {
	return fnc.Migration(
		func(ctx context.Context) error {
			return migrationFunc(ctx, DB)
		},
		fnc.WithSkip(2),
		fnc.WithSource(Source),
	)
}
