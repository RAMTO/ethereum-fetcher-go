package repository

import (
	"context"
	"errors"

	"ethereum-fetcher-go/internal/models"

	"gorm.io/gorm"
)

// transactionRepository implements TransactionRepository interface
type transactionRepository struct {
	*BaseRepository
}

// NewTransactionRepository creates a new transaction repository instance
func NewTransactionRepository(db *gorm.DB) TransactionRepository {
	return &transactionRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new transaction
func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	return r.DB.WithContext(ctx).Create(tx).Error
}

// GetByID retrieves a transaction by ID
func (r *transactionRepository) GetByID(ctx context.Context, id int) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.DB.WithContext(ctx).First(&tx, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// GetByHash retrieves a transaction by hash
func (r *transactionRepository) GetByHash(ctx context.Context, hash string) (*models.Transaction, error) {
	var tx models.Transaction
	err := r.DB.WithContext(ctx).Where("transaction_hash = ?", hash).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

// GetByBlockNumber retrieves transactions by block number
func (r *transactionRepository) GetByBlockNumber(ctx context.Context, blockNumber int) ([]*models.Transaction, error) {
	var txs []*models.Transaction
	err := r.DB.WithContext(ctx).Where("block_number = ?", blockNumber).Find(&txs).Error
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// GetByAddress retrieves transactions by address (from or to)
func (r *transactionRepository) GetByAddress(ctx context.Context, address string) ([]*models.Transaction, error) {
	var txs []*models.Transaction
	err := r.DB.WithContext(ctx).
		Where(`"from" = ? OR "to" = ?`, address, address).
		Find(&txs).Error
	if err != nil {
		return nil, err
	}
	return txs, nil
}

// Update updates an existing transaction
func (r *transactionRepository) Update(ctx context.Context, tx *models.Transaction) error {
	return r.DB.WithContext(ctx).Save(tx).Error
}

// Delete removes a transaction by ID
func (r *transactionRepository) Delete(ctx context.Context, id int) error {
	return r.DB.WithContext(ctx).Delete(&models.Transaction{}, id).Error
}
