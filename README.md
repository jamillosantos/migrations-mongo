# migrations-mongo

MongoDB migration target for [migrations](https://github.com/jamillosantos/migrations). It tracks which migrations have been applied using a MongoDB collection and provides distributed locking so multiple instances can safely run migrations concurrently.

## Installation

```bash
go get github.com/jamillosantos/migrations-mongo/v2
```

## Usage

```go
package main

import (
	"context"
	"log"

	"github.com/jamillosantos/migrations/v2"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	migrationsmongo "github.com/jamillosantos/migrations-mongo/v2"
)

func main() {
	ctx := context.Background()

	client, err := mongo.Connect(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	db := client.Database("myapp")

	source := migrations.NewMemorySource()
	// ... add migrations to source

	target, err := migrationsmongo.NewTarget(db)
	if err != nil {
		log.Fatal(err)
	}

	stats, err := migrations.Migrate(ctx, source, target)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("applied %d migrations", len(stats.Successful))
}
```

## Options

```go
target, err := migrationsmongo.NewTarget(db,
	migrationsmongo.CollectionName("_custom_migrations"), // default: "_migrations"
	migrationsmongo.LockTimeout(30 * time.Second),        // default: 10s
	migrationsmongo.OperationTimeout(30 * time.Second),   // default: 10s
)
```

## Writing migrations

Migrations are plain Go functions. The recommended naming convention uses a timestamp prefix as the migration ID:

```
migrations/
  ├─ 20230803201431_create_user_collection.go
  └─ 20230803204512_add_index_to_users_updated_at.go
     -------------- -----------------------------
          |                     |
          |                     └─ Description
          └─ ID (YYYYMMDDHHmmss)
```

Example migration:

```go
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
		Keys: bson.D{{"email", 1}},
		Options: options.Index().
			SetName("idx_unique_users_email").
			SetUnique(true),
	})
	return err
})
```

Check the full [example](internal/example) for a working setup with the migration helper pattern.

## Distributed locking

The target acquires a lock in a `<collection>_lock` collection before running migrations. This ensures that only one instance applies migrations at a time, even across multiple deployments. If the lock cannot be acquired within the configured timeout, it returns `ErrLockTimeout`.
