package models

import (
	"database/sql"
	"encoding/json"
)

// RentSession mirrors the structure of the "rent_sessions" table in PostgreSQL
type RentSession struct {
	ID                          int           `json:"id"`
	Renter                      int           `json:"renter"`
	NodeID                      sql.NullInt32 `json:"node_id"`
	Status                      string        `json:"status"`
	Platform                    string        `json:"platform"`
	TotalPrice                  float64       `json:"total_price"`
	RentStartTime               sql.NullTime  `json:"rent_start_time"`
	LastBalanceUpdateTimestamp  sql.NullTime  `json:"last_balance_update_timestamp"`
	RentStopTime                sql.NullTime  `json:"rent_stop_time"`
}

// MarshalJSON ensures that the time fields are properly handled for null values
func (r RentSession) MarshalJSON() ([]byte, error) {
	type Alias RentSession
	return json.Marshal(&struct {
		RentStartTime              interface{} `json:"rent_start_time,omitempty"`
		LastBalanceUpdateTimestamp interface{} `json:"last_balance_update_timestamp,omitempty"`
		RentStopTime               interface{} `json:"rent_stop_time,omitempty"`
		NodeID                     interface{} `json:"node_id,omitempty"`
		Alias
	}{
		RentStartTime:              NullTimeOrValue(r.RentStartTime),
		LastBalanceUpdateTimestamp: NullTimeOrValue(r.LastBalanceUpdateTimestamp),
		RentStopTime:               NullTimeOrValue(r.RentStopTime),
		NodeID:                     NullInt32OrValue(r.NodeID),
		Alias:                      (Alias)(r),
	})
}

// RentSessionInput is used for creating/updating RentSessions via JSON
type RentSessionInput struct {
	ID                          int     `json:"id,omitempty"`
	Renter                      int     `json:"renter,omitempty"`
	NodeID                      *int    `json:"node_id,omitempty"`
	Status                      string  `json:"status,omitempty"`
	Platform                    string  `json:"platform,omitempty"`
	TotalPrice                  float64 `json:"total_price,omitempty"`
	RentStartTime               *string `json:"rent_start_time,omitempty"`
	LastBalanceUpdateTimestamp  *string `json:"last_balance_update_timestamp,omitempty"`
	RentStopTime                *string `json:"rent_stop_time,omitempty"`
}

