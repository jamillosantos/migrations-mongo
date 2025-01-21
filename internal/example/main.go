package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jamillosantos/migrations/v2"
	"github.com/jamillosantos/migrations/v2/reporters"
	_ "github.com/jamillosantos/zapfancyencoder"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	migrationsmongo "github.com/jamillosantos/migrations-mongo"
	examplemigrations "github.com/jamillosantos/migrations-mongo/internal/example/migrations"
)

func main() {
	// Initialize the logger with the fancy encoding. This is not required at all. It is just for a better visualization.
	zapcfg := zap.NewDevelopmentConfig()
	zapcfg.Encoding = "fancy"
	logger, err := zapcfg.Build()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	// -----

	ctx := context.Background()

	// Start a new mongo connection
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://guest:guest@localhost:27017"))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to connect with mongo: %s\n", err.Error())
		os.Exit(1)
	}
	// -----

	db := client.Database("example")

	// Initialize the DB that will be used in the migrations.
	examplemigrations.DB = db

	// Create the target where the migrations will run.
	target, err := migrationsmongo.NewTarget(db)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to initialize target: %s\n", err)
		os.Exit(1)
	}

	reporter := reporters.NewZapReporter(logger) // If you do not use zap, you can implement your own reporter.

	// Run the migrations.
	_, err = migrations.Migrate(ctx, examplemigrations.Source, target, migrations.WithRunnerOptions(
		migrations.WithReporter(reporter),
	))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to migrate: %s\n", err)
		os.Exit(1)
	}
}
