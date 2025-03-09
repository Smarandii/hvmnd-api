package models

import (
	"time"
)

// CryptoPaymentTransaction represents a crypto deposit transaction
type CryptoPaymentTransaction struct {
	ID               int        `json:"id" db:"id"`
	UserID           int        `json:"user_id" db:"user_id"`
	DepositAddressID int        `json:"deposit_address_id" db:"deposit_address_id"`
	NetworkID        int        `json:"network_id" db:"network_id"`
	Amount           float64    `json:"amount" db:"amount"`
	TransactionHash  string     `json:"transaction_hash,omitempty" db:"transaction_hash"`
	Status           string     `json:"status" db:"status"`
	Confirmations    int        `json:"confirmations" db:"confirmations"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	ConfirmedAt      *time.Time `json:"confirmed_at,omitempty" db:"confirmed_at"`

	// Relations (not stored in DB)
	Network        *CryptoNetwork        `json:"network,omitempty" db:"-"`
	DepositAddress *CryptoDepositAddress `json:"deposit_address,omitempty" db:"-"`
}

// TableName returns the database table name for this model
func (CryptoPaymentTransaction) TableName() string {
	return "crypto_payment_transactions"
}

// Status constants for crypto payment transactions
const (
	TransactionStatusPending    = "pending"
	TransactionStatusConfirming = "confirming"
	TransactionStatusConfirmed  = "confirmed"
	TransactionStatusFailed     = "failed"
)
