package models

import "time"

type TgUserInteraction struct {
	ID         int       `json:"id"`
	TelegramID int64     `json:"telegram_id"`
	EventType  string    `json:"event_type"`
	EventData  string    `json:"event_data,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

type TgUserInteractionInput struct {
	TelegramID int64  `json:"telegram_id"` // required
	EventType  string `json:"event_type"`  // required (e.g. "message", "callback_query")
	EventData  string `json:"event_data,omitempty"`
}
