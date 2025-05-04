package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ------ ENUMS & Constants ------
// Chat status values – mirror the chat_status ENUM in the DB.
const (
	ChatStatusOpen   = "open"
	ChatStatusClosed = "closed"
)

// Sender values – mirror the message_sender ENUM in the DB.
const (
	SenderCustomer = "customer"
	SenderAgent    = "agent"
	SenderSystem   = "system"
)

// ------ MODELS ------

type SupportAgent struct {
	ID             int       `json:"id"`
	TelegramUserID int64     `json:"telegram_user_id"`
	Nickname       string    `json:"nickname"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
}

type SupportChat struct {
	ID             int           `json:"id"`
	CustomerID     int           `json:"customer_id"`
	AgentID        sql.NullInt32 `json:"agent_id"`
	TelegramChatID sql.NullInt64 `json:"telegram_chat_id"`
	Status         string        `json:"status"`
	OpenedAt       time.Time     `json:"opened_at"`
	ClosedAt       sql.NullTime  `json:"closed_at"`

	// useful expansions
	Agent *SupportAgent `json:"agent,omitempty"`
}

// custom JSON so that nullables render as null, not the zero‑value of the Go type.
func (c SupportChat) MarshalJSON() ([]byte, error) {
	type Alias SupportChat
	return json.Marshal(&struct {
		AgentID        interface{} `json:"agent_id"`
		TelegramChatID interface{} `json:"telegram_chat_id"`
		ClosedAt       interface{} `json:"closed_at"`
		Alias
	}{
		AgentID:        NullInt32OrValue(c.AgentID),
		TelegramChatID: NullInt64OrValue(c.TelegramChatID),
		ClosedAt:       NullTimeOrValue(c.ClosedAt),
		Alias:          (Alias)(c),
	})
}

type SupportMessage struct {
	ID                  int           `json:"id"`
	ChatID              int           `json:"chat_id"`
	Sender              string        `json:"sender"`
	SenderCustomerID    sql.NullInt32 `json:"sender_customer_id"`
	SenderAgentID       sql.NullInt32 `json:"sender_agent_id"`
	TelegramMessageID   sql.NullInt64 `json:"telegram_message_id"`
	Content             string        `json:"content"`
	CreatedAt           time.Time     `json:"created_at"`
	DeliveredToTelegram bool          `json:"delivered_to_telegram"`
	DeliveredToWeb      bool          `json:"delivered_to_web"`
	ReadByCustomer      bool          `json:"read_by_customer"`
	ReadByAgent         bool          `json:"read_by_agent"`
}

func (m SupportMessage) MarshalJSON() ([]byte, error) {
	type Alias SupportMessage
	return json.Marshal(&struct {
		SenderCustomerID  interface{} `json:"sender_customer_id"`
		SenderAgentID     interface{} `json:"sender_agent_id"`
		TelegramMessageID interface{} `json:"telegram_message_id"`
		Alias
	}{
		SenderCustomerID:  NullInt32OrValue(m.SenderCustomerID),
		SenderAgentID:     NullInt32OrValue(m.SenderAgentID),
		TelegramMessageID: NullInt64OrValue(m.TelegramMessageID),
		Alias:             (Alias)(m),
	})
}
