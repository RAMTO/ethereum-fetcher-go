package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ethereum-fetcher-go/internal/models"
)

type userTransactionRepository struct {
	db *gorm.DB
}

// NewUserTransactionRepository creates a new UserTransactionRepository
func NewUserTransactionRepository(db *gorm.DB) UserTransactionRepository {
	return &userTransactionRepository{
		db: db,
	}
}

// Create creates a new user-transaction association
func (r *userTransactionRepository) Create(ctx context.Context, userID int, transactionHash string) error {
	userTransaction := &models.UserTransaction{
		UserID:          userID,
		TransactionHash: transactionHash,
		FetchedAt:       time.Now(),
	}

	return r.db.WithContext(ctx).Create(userTransaction).Error
}

// GetByTransactionHashAndUserId retrieves a user transaction by transaction ID and user ID
func (r *userTransactionRepository) GetByTransactionHashAndUserId(ctx context.Context, transactionHash string, userID int) (*models.UserTransaction, error) {
	fmt.Println("transactionHash", transactionHash)
	var userTransaction models.UserTransaction

	fmt.Println("transactionHash", transactionHash)
	fmt.Println("userID", userID)

	result := r.db.WithContext(ctx).
		Where("transaction_hash = ? AND user_id = ?", transactionHash, userID).
		First(&userTransaction)

	fmt.Println("result", result)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return &userTransaction, nil
}

func (r *userTransactionRepository) GetTransactionsByUserId(ctx context.Context, userID int) ([]*models.UserTransaction, error) {
	var userTransactions []*models.UserTransaction

	return userTransactions, r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&userTransactions).Error
}
