package models

import "time"

type WebAppSessionToken struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
}
