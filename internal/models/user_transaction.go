package models

import "time"

// UserTransaction represents the many-to-many relationship between users and transactions
// It includes additional metadata about when the transaction was fetched by the user
type UserTransaction struct {
	ID              int       `json:"id" gorm:"primaryKey"`
	UserID          int       `json:"user_id" gorm:"foreignKey:UserID"`
	TransactionHash string    `json:"transaction_hash"`
	FetchedAt       time.Time `json:"fetched_at" gorm:"not null;default:current_timestamp"`
}
