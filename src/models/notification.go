package models

import (
	"database/sql"
	"encoding/json"
)

type Notification struct {
	ID                   int          `json:"id"`
	UserId               int          `json:"user_id"`
	NotificationText     string       `json:"notification_text"`
	NotificationPlatform string       `json:"notification_platform"`
	CreatedAt            sql.NullTime `json:"created_at"`
	UpdatedAt            sql.NullTime `json:"updated_at"`
	IsRead               bool         `json:"is_read"`
	IsSent               bool         `json:"is_sent"`
}

func (n Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		CreatedAt interface{} `json:"created_at,omitempty"`
		UpdatedAt interface{} `json:"updated_at,omitempty"`
		Alias
	}{
		CreatedAt: NullTimeOrValue(n.CreatedAt),
		UpdatedAt: NullTimeOrValue(n.UpdatedAt),
		Alias:     (Alias)(n),
	})
}

type NotificationInput struct {
	ID                   int    `json:"id,omitempty"`
	UserId               int    `json:"user_id,omitempty"`
	NotificationText     string `json:"notification_text,omitempty"`
	NotificationPlatform string `json:"notification_platform,omitempty"`
	IsRead               bool   `json:"is_read,omitempty"`
	IsSent               bool   `json:"is_sent,omitempty"`
}

