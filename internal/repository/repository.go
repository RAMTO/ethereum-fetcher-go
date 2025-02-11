package repository

import (
	"context"

	"ethereum-fetcher-go/internal/models"

	"gorm.io/gorm"
)

// Repository defines the base interface for all repositories
type Repository interface {
	// Common methods that all repositories should implement
	Close() error
}

// BaseRepository provides common functionality for all repositories
type BaseRepository struct {
	DB *gorm.DB
}

// NewBaseRepository creates a new base repository instance
func NewBaseRepository(db *gorm.DB) *BaseRepository {
	return &BaseRepository{DB: db}
}

// Close closes the database connection
func (r *BaseRepository) Close() error {
	sqlDB, err := r.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// UserRepository defines the interface for user-related operations
type UserRepository interface {
	Repository
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id int) error
}

// TransactionRepository defines the interface for transaction-related operations
type TransactionRepository interface {
	Repository
	Create(ctx context.Context, tx *models.Transaction) error
	GetByID(ctx context.Context, id int) (*models.Transaction, error)
	GetByHash(ctx context.Context, hash string) (*models.Transaction, error)
	GetByBlockNumber(ctx context.Context, blockNumber int) ([]*models.Transaction, error)
	GetByAddress(ctx context.Context, address string) ([]*models.Transaction, error)
	Update(ctx context.Context, tx *models.Transaction) error
	Delete(ctx context.Context, id int) error
}
