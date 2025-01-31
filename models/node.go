package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Node struct {
	ID                         int             `json:"id"`
	OldID                      sql.NullInt32   `json:"old_id"`
	AnyDeskAddress             string          `json:"any_desk_address"`
	AnyDeskPassword            string          `json:"any_desk_password"`
	Status                     string          `json:"status"`
	Software                   sql.NullString  `json:"software"`
	Price                      sql.NullFloat64 `json:"price"`
	Renter                     sql.NullInt16   `json:"renter"`
	RentStartTime              sql.NullTime    `json:"rent_start_time"`
	LastBalanceUpdateTimestamp sql.NullTime    `json:"last_balance_update_timestamp"`
	CPU                        sql.NullString  `json:"cpu"`
	GPU                        sql.NullString  `json:"gpu"`
	OtherSpecs                 sql.NullString  `json:"other_specs"`
	Licenses                   sql.NullString  `json:"licenses"`
	MachineID                  string          `json:"machine_id"`
}

func (n Node) MarshalJSON() ([]byte, error) {
	type Alias Node
	return json.Marshal(&struct {
		OldID                      interface{} `json:"old_id"`
		Software                   interface{} `json:"software"`
		Price                      interface{} `json:"price"`
		Renter                     interface{} `json:"renter"`
		RentStartTime              interface{} `json:"rent_start_time"`
		LastBalanceUpdateTimestamp interface{} `json:"last_balance_update_timestamp"`
		CPU                        interface{} `json:"cpu"`
		GPU                        interface{} `json:"gpu"`
		OtherSpecs                 interface{} `json:"other_specs"`
		Licenses                   interface{} `json:"licenses"`
		MachineID                  interface{} `json:"machine_id"`
		Alias
	}{
		OldID:                      NullInt32OrValue(n.OldID),
		Software:                   NullStringOrValue(n.Software),
		Price:                      NullFloat64OrValue(n.Price),
		Renter:                     NullInt16OrValue(n.Renter),
		RentStartTime:              NullTimeOrValue(n.RentStartTime),
		LastBalanceUpdateTimestamp: NullTimeOrValue(n.LastBalanceUpdateTimestamp),
		CPU:                        NullStringOrValue(n.CPU),
		GPU:                        NullStringOrValue(n.GPU),
		OtherSpecs:                 NullStringOrValue(n.OtherSpecs),
		Licenses:                   NullStringOrValue(n.Licenses),
		Alias:                      (Alias)(n),
	})
}

type NodeInput struct {
	ID                         *int16     `json:"id,omitempty"`
	OldID                      *int16     `json:"old_id,omitempty"`
	AnyDeskAddress             *string    `json:"any_desk_address,omitempty"`
	AnyDeskPassword            *string    `json:"any_desk_password,omitempty"`
	Status                     *string    `json:"status,omitempty"`
	Software                   *string    `json:"software,omitempty"`
	Price                      *float64   `json:"price,omitempty"`
	Renter                     *int16     `json:"renter,omitempty"`
	RentStartTime              *time.Time `json:"rent_start_time,omitempty"`
	LastBalanceUpdateTimestamp *time.Time `json:"last_balance_update_timestamp,omitempty"`
	CPU                        *string    `json:"cpu,omitempty"`
	GPU                        *string    `json:"gpu,omitempty"`
	OtherSpecs                 *string    `json:"other_specs,omitempty"`
	Licenses                   *string    `json:"licenses,omitempty"`
	MachineID                  *string    `json:"machine_id,omitempty"`
}
