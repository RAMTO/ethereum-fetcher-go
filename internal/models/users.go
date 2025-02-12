package models

import "time"

type User struct {
	ID           int           `json:"id" gorm:"primaryKey"`
	Username     string        `json:"username" gorm:"unique;not null"`
	Password     string        `json:"password" gorm:"not null"`
	CreatedAt    time.Time     `json:"created_at" gorm:"not null"`
	Transactions []Transaction `json:"transactions" gorm:"many2many:user_transactions;"`
}
