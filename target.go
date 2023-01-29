package migrationsmongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jamillosantos/migrations"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrMissingSource = errors.New("source is required")
	ErrLockTimeout   = errors.New("timeout while trying to lock the database")
)

type migrationModel struct {
	ID string `bson:"_id"`
}

type Target struct {
	source             migrations.Source
	db                 *mongo.Database
	collectionName     string
	lockCollectionName string
	lockTimeout        time.Duration
	operationTimeout   time.Duration
}

type Option func(target *Target) error

// DefaultMigrationsCollectionName is the default collection name to be used to store the migrations.
const DefaultMigrationsCollectionName = "_migrations"

func NewTarget(source migrations.Source, db *mongo.Database, options ...Option) (*Target, error) {
	if source == nil {
		return nil, ErrMissingSource
	}
	target := &Target{
		source:           source,
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

func (t *Target) Create() error {
	err := t.db.CreateCollection(context.Background(), t.collectionName)
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
	we, ok := err.(mongo.CommandError)
	return ok && we.Code == 48
}

func (t *Target) Destroy() error {
	// TODO if the collection don't exist, don't fail.
	return t.db.Collection(t.collectionName).Drop(context.Background())
}

var (
	sortByPK = options.Find().SetSort(bson.D{{"_id", 1}})
)

func (t *Target) Current() (migrations.Migration, error) {
	list, err := t.Done()
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, migrations.ErrNoCurrentMigration
	}
	return list[len(list)-1], nil
}

func (t *Target) Done() ([]migrations.Migration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.operationTimeout)
	defer cancel()

	rs, err := t.db.Collection(t.collectionName).Find(ctx, bson.D{}, sortByPK)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rs.Close(ctx)
	}()

	result := make([]migrations.Migration, 0)
	for rs.Next(ctx) {
		var m migrationModel
		if err := rs.Decode(&m); err != nil {
			return nil, err
		}
		migration, err := t.source.ByID(m.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, migration)
	}
	return result, nil
}

func (t *Target) Lock() (migrations.Unlocker, error) {
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

func (t *Target) Add(migration migrations.Migration) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.operationTimeout)
	defer cancel()

	_, err := t.db.Collection(t.collectionName).InsertOne(ctx, migrationModel{
		ID: migration.ID(),
	})
	if err != nil {
		return fmt.Errorf("failed adding migration to the executed list: %w", err)
	}
	return nil
}

func (t *Target) Remove(migration migrations.Migration) error {
	ctx, cancel := context.WithTimeout(context.Background(), t.operationTimeout)
	defer cancel()
	_, err := t.db.Collection(t.collectionName).DeleteOne(ctx, migrationModel{migration.ID()})
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
