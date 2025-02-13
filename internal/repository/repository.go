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
	Create(ctx context.Context, user *models.User) (*models.User, error)
	GetByID(ctx context.Context, id int) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
}

// TransactionRepository defines the interface for transaction-related operations
type TransactionRepository interface {
	Repository
	Create(ctx context.Context, tx *models.Transaction) (*models.Transaction, error)
	GetByID(ctx context.Context, id int) (*models.Transaction, error)
	GetAll(ctx context.Context) ([]*models.Transaction, error)
	GetByHash(ctx context.Context, hash string) (*models.Transaction, error)
	GetByHashes(ctx context.Context, hashes []string) ([]*models.Transaction, error)
}

type UserTransactionRepository interface {
	Repository
	Create(ctx context.Context, userID int, transactionHash string) (*models.UserTransaction, error)
	GetByTransactionHashAndUserId(ctx context.Context, transactionHash string, userID int) (*models.UserTransaction, error)
	GetTransactionsByUserId(ctx context.Context, userID int) ([]*models.UserTransaction, error)
}
