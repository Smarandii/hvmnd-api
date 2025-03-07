package models

import (
	"database/sql"
	"encoding/json"
)

type Token struct {
	ID         int          `json:"id"`
	Token      string       `json:"token"`
	UserId     int          `json:"user_id"`
	TelegramID int          `json:"telegram_id"`
	Status     string       `json:"status"`
	CreatedAt  sql.NullTime `json:"created_at"`
}

func (t Token) MarshalJSON() ([]byte, error) {
	type Alias Token
	return json.Marshal(&struct {
		ID         interface{} `json:"id"`
		Token      interface{} `json:"token"`
		UserId     interface{} `json:"user_id"`
		TelegramID interface{} `json:"telegram_id"`
		Status     interface{} `json:"status"`
		CreatedAt  interface{} `json:"created_at"`
		Alias
	}{
		ID:         t.ID,
		Token:      t.Token,
		UserId:     t.UserId,
		TelegramID: t.TelegramID,
		Status:     t.Status,
		CreatedAt:  NullTimeOrValue(t.CreatedAt),
		Alias:      (Alias)(t),
	})
}

type TokenInput struct {
	UserId     int    `json:"user_id"`
	TelegramID int    `json:"telegram_id"`
	Status     string `json:"status"`
}
