package migrationsmongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type lockerPK struct {
	ID string `bson:"_id"`
}

type lockerModel struct {
	ID     string  `bson:"_id"`
	LockID *string `bson:"lock_id"`
}

type mgLocker struct {
	db             *mongo.Database
	collectionName string
	lockID         string
}

func (p *mgLocker) Unlock() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10) // TODO Make this configurable
	defer cancel()
	lockCollection := p.db.Collection(p.collectionName)
	err := lockCollection.FindOneAndUpdate(ctx,
		lockerPK{"lock"},
		bson.D{
			{
				"$set",
				bson.D{
					{"lock_id", nil},
				},
			},
		},
	).Err()
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		return nil
	case err != nil:
		return fmt.Errorf("failed unlocking migration: %w", err)
	}
	return nil
}
