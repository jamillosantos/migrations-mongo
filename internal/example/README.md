# Example

## How to run

```bash
cd internal/example
docker-compose up -d
go run main.go
```

This should output:

```
      Level: INFO
    Message: migration plan with 2 migrations
  Timestamp: 2023-08-03 20:46:46 -03:00
     Fields: 
      └─ plan: 
          ├─ [0]: 20230803201431_create user collection (do)
          └─ [1]: 20230803204512_add index to users updated at (do)
------------------------------------
      Level: INFO
    Message: migration 20230803201431_create user collection (do)
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: migration 20230803201431_create user collection (do) successfully applied
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: migration 20230803204512_add index to users updated at (do)
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: migration 20230803204512_add index to users updated at (do) successfully applied
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: SUCCESS: migration has finished with 2 successes and 0 failures
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: migration 20230803201431_create user collection (do) was applied
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
      Level: INFO
    Message: migration 20230803204512_add index to users updated at (do) was applied
  Timestamp: 2023-08-03 20:46:46 -03:00
------------------------------------
```

> The above are zap logs using the [zapfancyencoder](github.com/jamillosantos/zapfancyencoder).

## Migrations

```
└─ internal/example/migrations:
    ├─ 20230803201431_create_user_collection.go
    └─ 20230803204512_add_index_to_users_updated_at.go
       -------------- -----------------------------
            |                     |
            |                     └─ Description.
            └─ ID
```

* __ID__: The ID of the migration. We recommend the use of the timestamp in the format `YYYYMMDDHHmmss`. But, in 
  reality, it can be any `int64` number.
* __Description__: The description of the migration. This is used to identify the migration in the logs for humans.

### Migration code

```go
package migrations

import (
	"context"

	. "github.com/jamillosantos/migrations-fnc" // <- provides the Migration function.
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var _ = Migration(func(ctx context.Context) error {
	// DB should be a global variable initialized before the migration runs.
	
	// Get the collection.
	c := DB.Collection("users")
	
	// Create the index.
	_, err := c.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]int{"updated_at": 1},
		Options: (&options.IndexOptions{}).
			SetName("idx_users_updated_at"),
	})
	return err
})
```

The migrations in Mongo use the [migrations-fnc](github.com/jamillosantos/migrations-fnc) package. The `migration-fnc` 
enables functions to be run as migrations.

In the case of Mongo, we will run the Go code to create or drop indexes, collections, etc.

