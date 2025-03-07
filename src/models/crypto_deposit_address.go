package models

import (
	"time"
)

// CryptoDepositAddress represents a user's deposit address for a specific network
type CryptoDepositAddress struct {
	ID        int        `json:"id" db:"id"`
	UserID    int        `json:"user_id" db:"user_id"`
	NetworkID int        `json:"network_id" db:"network_id"`
	Address   string     `json:"address" db:"address"`
	IsUsed    bool       `json:"is_used" db:"is_used"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UsedAt    *time.Time `json:"used_at,omitempty" db:"used_at"`

	// Relations (not stored in DB)
	Network *CryptoNetwork `json:"network,omitempty" db:"-"`
}

// TableName returns the database table name for this model
func (CryptoDepositAddress) TableName() string {
	return "crypto_deposit_addresses"
}
