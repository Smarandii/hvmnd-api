package models

import (
	"database/sql"
	"encoding/json"
)

type User struct {
	ID           int            `json:"id"`
	TelegramID   int            `json:"telegram_id"`
	TotalSpent   float64        `json:"total_spent"`
	Balance      float64        `json:"balance"`
	FirstName    sql.NullString `json:"-"`
	LastName     sql.NullString `json:"-"`
	Username     sql.NullString `json:"-"`
	LanguageCode sql.NullString `json:"-"`
	Banned       sql.NullBool   `json:"-"`
}

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		FirstName    interface{} `json:"first_name"`
		LastName     interface{} `json:"last_name"`
		Username     interface{} `json:"username"`
		LanguageCode interface{} `json:"language_code"`
		Banned       interface{} `json:"banned"`
		Alias
	}{
		FirstName:    NullStringOrValue(u.FirstName),
		LastName:     NullStringOrValue(u.LastName),
		Username:     NullStringOrValue(u.Username),
		LanguageCode: NullStringOrValue(u.LanguageCode),
		Banned:       NullBoolOrValue(u.Banned),
		Alias:        (Alias)(u),
	})
}

type UserInput struct {
	TelegramID   int      `json:"telegram_id"`
	TotalSpent   *float64 `json:"total_spent,omitempty"` // Use pointer to detect if the field is present
	Balance      *float64 `json:"balance,omitempty"`     // Same here
	FirstName    *string  `json:"first_name,omitempty"`
	LastName     *string  `json:"last_name,omitempty"`
	Username     *string  `json:"username,omitempty"`
	LanguageCode *string  `json:"language_code,omitempty"`
	Banned       *bool    `json:"banned,omitempty"`
}
