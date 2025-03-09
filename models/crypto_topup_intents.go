package models

import (
	"time"
)

type CryptoTopupIntent struct {
	ID              int        `json:"id" db:"id"`
	UserID          int        `json:"user_id" db:"user_id"`
	NetworkID       int        `json:"network_id" db:"network_id"`
	Amount          float64    `json:"amount" db:"amount"`
	Status          string     `json:"status" db:"status"`
	TransactionHash string     `json:"transaction_hash,omitempty" db:"transaction_hash"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	ConfirmedAt     *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty" db:"expires_at"`

	// Relations (not stored in DB)
	Network *CryptoNetwork `json:"network,omitempty" db:"-"`
}

// TableName returns the database table name for this model
func (CryptoTopupIntent) TableName() string {
	return "crypto_topup_intents"
}

// Status constants for crypto payment transactions
const (
	TopupIntentStatusPending    = "pending"
	TopupIntentStatusConfirming = "confirming"
	TopupIntentStatusConfirmed  = "confirmed"
	TopupIntentStatusFailed     = "failed"
	TopupIntentStatusCancelled  = "cancelled"
)
