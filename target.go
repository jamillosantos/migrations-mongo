package migrationsmongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jamillosantos/migrations/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMissingSource = errors.New("source is required")
	ErrLockTimeout   = errors.New("timeout while trying to lock the database")
)

type migrationModel struct {
	ID    string `bson:"_id"`
	Dirty bool   `bson:"dirty"`
}

type Target struct {
	db                 *mongo.Database
	collectionName     string
	lockCollectionName string
	lockTimeout        time.Duration
	operationTimeout   time.Duration
}

type Option func(target *Target) error

// DefaultMigrationsCollectionName is the default collection name to be used to store the migrations.
const DefaultMigrationsCollectionName = "_migrations"

func NewTarget(db *mongo.Database, options ...Option) (*Target, error) {
	target := &Target{
		db:               db,
		collectionName:   DefaultMigrationsCollectionName,
		lockTimeout:      DefaultLockTimeout,
		operationTimeout: DefaultOperationTimeout,
	}
	for _, opt := range options {
		err := opt(target)
		if err != nil {
			return nil, err
		}
	}
	target.lockCollectionName = fmt.Sprintf("%s_lock", target.collectionName)
	return target, nil
}

func (t *Target) Create(ctx context.Context) error {
	err := t.db.CreateCollection(ctx, t.collectionName)
	switch {
	case isCollectionAlreadyExists(err):
		return nil
	case err != nil:
		return fmt.Errorf("error creating collection: %w", err)
	}
	return nil
}

func isCollectionAlreadyExists(err error) bool {
	if err == nil {
		return false
	}
	var we mongo.CommandError
	ok := errors.As(err, &we)
	return ok && we.Code == 48
}

func (t *Target) Destroy(ctx context.Context) error {
	// TODO if the collection don't exist, don't fail.
	return t.db.Collection(t.collectionName).Drop(ctx)
}

var (
	sortByPK = options.Find().SetSort(bson.D{{"_id", 1}})
)

func (t *Target) Current(ctx context.Context) (string, error) {
	list, err := t.Done(ctx)
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", migrations.ErrNoCurrentMigration
	}
	return list[len(list)-1], nil
}

func (t *Target) Done(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.operationTimeout)
	defer cancel()

	rs, err := t.db.Collection(t.collectionName).Find(ctx, bson.D{}, sortByPK)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rs.Close(ctx)
	}()

	result := make([]string, 0)
	for rs.Next(ctx) {
		var m migrationModel
		if err := rs.Decode(&m); err != nil {
			return nil, err
		}
		result = append(result, m.ID)
	}
	return result, nil
}

func (t *Target) Lock(ctx context.Context) (migrations.Unlocker, error) {
	lockID, err := t.generateLockID()
	if err != nil {
		return nil, fmt.Errorf("failed locking database: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), t.lockTimeout)
	defer cancel()

	lockCollection := t.db.Collection(t.lockCollectionName)
LockStart:
	_, err = lockCollection.UpdateOne(ctx,
		bson.D{
			{"_id", "lock"},
			{"lock_id", bson.D{{"$type", 10}}},
		},
		bson.D{{"$set", lockerModel{
			ID:     "lock",
			LockID: &lockID,
		}}},
		options.Update().SetUpsert(true),
	)
	switch {
	case errors.Is(err, mongo.ErrNoDocuments) || mongo.IsDuplicateKeyError(err):
		select {
		case <-ctx.Done(): // if times out
			return nil, ErrLockTimeout
		case <-time.After(time.Second):
		}
		goto LockStart
	case err != nil:
		return nil, fmt.Errorf("failed locking migration: %w", err)
	}
	return &mgLocker{db: t.db, collectionName: t.lockCollectionName, lockID: lockID}, nil
}

func (t *Target) Add(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, t.operationTimeout)
	defer cancel()

	_, err := t.db.Collection(t.collectionName).InsertOne(ctx, migrationModel{
		ID:    id,
		Dirty: true,
	})
	if err != nil {
		return fmt.Errorf("failed adding migration to the executed list: %w", err)
	}
	return nil
}

func (t *Target) FinishMigration(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, t.operationTimeout)
	defer cancel()

	_, err := t.db.Collection(t.collectionName).UpdateOne(ctx,
		bson.D{{"_id", id}},
		bson.D{{"$set", bson.D{{"dirty", false}}}},
	)
	if err != nil {
		return fmt.Errorf("failed finishing migration: %w", err)
	}
	return nil
}

func (t *Target) StartMigration(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, t.operationTimeout)
	defer cancel()

	_, err := t.db.Collection(t.collectionName).UpdateOne(ctx,
		bson.D{{"_id", id}},
		bson.D{{"$set", bson.D{{"dirty", true}}}},
	)
	if err != nil {
		return fmt.Errorf("failed starting migration: %w", err)
	}
	return nil
}

func (t *Target) Remove(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.operationTimeout)
	defer cancel()
	_, err := t.db.Collection(t.collectionName).DeleteOne(ctx, bson.D{{"_id", id}})
	if err != nil {
		return fmt.Errorf("failed removing migration from the executed list: %w", err)
	}
	return nil
}

func (t *Target) generateLockID() (string, error) {
	u, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
