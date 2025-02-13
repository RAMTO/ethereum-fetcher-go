package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"ethereum-fetcher-go/internal/models"
)

type userTransactionRepository struct {
	*BaseRepository
}

// NewUserTransactionRepository creates a new UserTransactionRepository
func NewUserTransactionRepository(db *gorm.DB) UserTransactionRepository {
	return &userTransactionRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new user-transaction association
func (r *userTransactionRepository) Create(ctx context.Context, userID int, transactionHash string) (*models.UserTransaction, error) {
	userTransaction := &models.UserTransaction{
		UserID:          userID,
		TransactionHash: transactionHash,
		FetchedAt:       time.Now(),
	}

	err := r.DB.WithContext(ctx).Create(userTransaction).Error

	if err != nil {
		return nil, err
	}

	return userTransaction, nil
}

// GetByTransactionHashAndUserId retrieves a user transaction by transaction ID and user ID
func (r *userTransactionRepository) GetByTransactionHashAndUserId(ctx context.Context, transactionHash string, userID int) (*models.UserTransaction, error) {
	var userTransaction models.UserTransaction

	err := r.DB.WithContext(ctx).
		Where("transaction_hash = ? AND user_id = ?", transactionHash, userID).
		First(&userTransaction).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &userTransaction, nil
}

func (r *userTransactionRepository) GetTransactionsByUserId(ctx context.Context, userID int) ([]*models.UserTransaction, error) {
	var userTransactions []*models.UserTransaction

	err := r.DB.WithContext(ctx).Where("user_id = ?", userID).Find(&userTransactions).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return userTransactions, nil
}
