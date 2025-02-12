package models

type Transaction struct {
	ID                int    `json:"id" gorm:"primaryKey"`
	TransactionHash   string `json:"transactionHash" gorm:"unique;not null"`
	TransactionStatus int    `json:"transactionStatus"`
	BlockHash         string `json:"blockHash"`
	BlockNumber       int    `json:"blockNumber"`
	From              string `json:"from"`
	To                string `json:"to"`
	ContractAddress   string `json:"contractAddress"`
	LogsCount         int    `json:"logsCount"`
	Input             string `json:"input"`
	Value             int    `json:"value"`
	Users             []User `json:"users" gorm:"many2many:user_transactions;"`
}
