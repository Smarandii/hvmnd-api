package models

import (
	"time"
)

type Payment struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	Amount   float64   `json:"amount"`
	Status   string    `json:"status"`
	Datetime time.Time `json:"datetime"`
}
