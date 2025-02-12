package models

import "time"

// UserTransaction represents the many-to-many relationship between users and transactions
// It includes additional metadata about when the transaction was fetched by the user
type UserTransaction struct {
	UserID        int       `json:"user_id" gorm:"primaryKey"`
	TransactionID int       `json:"transaction_id" gorm:"primaryKey"`
	FetchedAt     time.Time `json:"fetched_at" gorm:"not null;default:current_timestamp"`
}
