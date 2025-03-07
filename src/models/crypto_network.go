package models

import (
	"time"
)

// CryptoNetwork represents a blockchain network and token configuration
type CryptoNetwork struct {
	ID              int       `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	TokenSymbol     string    `json:"token_symbol" db:"token_symbol"`
	ContractAddress string    `json:"contract_address,omitempty" db:"contract_address"`
	NetworkFee      float64   `json:"network_fee" db:"network_fee"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// TableName returns the database table name for this model
func (CryptoNetwork) TableName() string {
	return "crypto_networks"
}
