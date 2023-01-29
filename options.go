package migrationsmongo

import (
	"time"
)

// CollectionName sets the collection name to be used to store the migrations. The default value is stored in the global
// variable DefaultMigrationsCollectionName.
func CollectionName(collectionName string) Option {
	return func(target *Target) error {
		if collectionName != "" {
			target.collectionName = collectionName
		}
		return nil
	}
}

var (
	// DefaultLockTimeout is the default timeout to be used when trying to lock the database.
	DefaultLockTimeout = 10 * time.Second
)

// LockTimeout sets the timeout to be used when trying to lock the database. The default value is read from DefaultLockTimeout.
func LockTimeout(timeout time.Duration) Option {
	return func(target *Target) error {
		target.lockTimeout = timeout
		return nil
	}
}

var (
	// DefaultOperationTimeout is the default timeout to be used when performing operations in the database. Eg; Add the
	// migration document to the collection.
	DefaultOperationTimeout = 10 * time.Second
)

// OperationTimeout sets the timeout to be used when performing operations in the database. The default value is read from DefaultOperationTimeout.
func OperationTimeout(timeout time.Duration) Option {
	return func(target *Target) error {
		target.operationTimeout = timeout
		return nil
	}
}
