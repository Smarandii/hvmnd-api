package models

import (
	"database/sql"
	"encoding/json"
	"hvmnd/api/utils"
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
}

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		FirstName    interface{} `json:"first_name"`
		LastName     interface{} `json:"last_name"`
		Username     interface{} `json:"username"`
		LanguageCode interface{} `json:"language_code"`
		Alias
	}{
		FirstName:    utils.NullStringOrValue(u.FirstName),
		LastName:     utils.NullStringOrValue(u.LastName),
		Username:     utils.NullStringOrValue(u.Username),
		LanguageCode: utils.NullStringOrValue(u.LanguageCode),
		Alias:        (Alias)(u),
	})
}
