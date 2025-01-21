package migrationsmongo

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/jamillosantos/migrations/v2"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestNewTarget(t *testing.T) {
	t.Run("should run migrations", func(t *testing.T) {
		db := createMongoClient(t)

		ctx := context.Background()

		source := migrations.NewMemorySource()
		target, err := NewTarget(db)
		require.NoError(t, err, "failed creating target")

		execution := make([]string, 0, 3)
		var executionL sync.Mutex
		m1 := migrations.NewMigration("1", "m1", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "1")
			return nil
		}, nil)
		m2 := migrations.NewMigration("2", "m2", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "2")
			return nil
		}, nil)
		m3 := migrations.NewMigration("3", "m3", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "3")
			return nil
		}, nil)
		require.NoError(t, source.Add(ctx, m1))
		require.NoError(t, source.Add(ctx, m2))
		require.NoError(t, source.Add(ctx, m3))

		stats, err := migrations.Migrate(ctx, source, target)
		require.NoError(t, err, "failed migrating")
		require.Len(t, stats.Errored, 0)
		require.Len(t, stats.Successful, 3)

		mongoMigrations := make([]migrationModel, 0)
		r, err := db.Collection(DefaultMigrationsCollectionName).Find(context.Background(), bson.D{})
		require.NoError(t, err)
		require.NoError(t, r.All(context.Background(), &mongoMigrations))
		require.Len(t, mongoMigrations, 3)
		require.Equal(t, "1", mongoMigrations[0].ID)
		require.Equal(t, "2", mongoMigrations[1].ID)
		require.Equal(t, "3", mongoMigrations[2].ID)

		require.Equal(t, []string{"1", "2", "3"}, execution)

		stats, err = migrations.Migrate(ctx, source, target)
		require.NoError(t, err, "failed migrating")
		require.Len(t, stats.Errored, 0)
		require.Len(t, stats.Successful, 0)
		require.Equal(t, []string{"1", "2", "3"}, execution)
	})

	t.Run("should run one migration at time", func(t *testing.T) {
		execution := make([]string, 0, 3)
		var executionL sync.Mutex
		m1 := migrations.NewMigration("1", "m1", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "1")
			return nil
		}, nil)
		m2 := migrations.NewMigration("2", "m2", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "2")
			return nil
		}, nil)
		m3 := migrations.NewMigration("3", "m3", func(ctx context.Context) error {
			executionL.Lock()
			defer executionL.Unlock()
			execution = append(execution, "3")
			return nil
		}, nil)

		db := createMongoClient(t)

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx := context.Background()

				source := migrations.NewMemorySource()
				require.NoError(t, source.Add(ctx, m1))
				require.NoError(t, source.Add(ctx, m2))
				require.NoError(t, source.Add(ctx, m3))

				target, err := NewTarget(db, LockTimeout(time.Second*10))
				require.NoError(t, err, "failed creating target")

				_, err = migrations.Migrate(ctx, source, target)
				require.NoError(t, err, "failed migrating")
			}()
		}

		wg.Wait()
		require.Equal(t, []string{"1", "2", "3"}, execution)
	})
}

func createMongoClient(t *testing.T) *mongo.Database {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	require.NoError(t, err, "failed connecting to docker")

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("mongo", "latest", []string{})
	require.NoError(t, err, "failed starting mongo")
	t.Cleanup(func() {
		_ = resource.Close()
	})

	client, err := mongo.NewClient(options.Client().SetHosts([]string{resource.GetHostPort("27017/tcp")}))
	require.NoError(t, err, "failed creating mongo client")

	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return client.Connect(ctx) == nil
	}, time.Minute, time.Second, "mongo is not ready")

	db := client.Database("migration_test")
	t.Cleanup(func() {
		client.Disconnect(context.Background())
	})
	return db
}
