package migrations

import (
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	// DB is the global database connection that will be used by the migrations.
	DB *mongo.Database
)
