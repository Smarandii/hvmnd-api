package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"net/http"
)

func GetNodes(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}

	renter := r.URL.Query().Get("renter")
	status := r.URL.Query().Get("status")
	anyDeskAddress := r.URL.Query().Get("anydesk_address")
	software := r.URL.Query().Get("software")

	query := `
		SELECT 
		id, old_id, any_desk_address, 
		any_desk_password, status, software, 
		price, renter, rent_start_time, 
		last_balance_update_timestamp, 
		cpu, gpu, other_specs, licenses, 
		machine_id FROM nodes WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}

	if renter != "" {
		query += fmt.Sprintf(" AND renter = $%d", argIndex)
		args = append(args, renter)
		argIndex++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if anyDeskAddress != "" {
		query += fmt.Sprintf(" AND any_desk_address = $%d", argIndex)
		args = append(args, anyDeskAddress)
		argIndex++
	}

	if software != "" {
		query += fmt.Sprintf(" AND software ILIKE $%d", argIndex)
		args = append(args, "%"+software+"%")
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var nodes []models.Node
	for rows.Next() {
		var node models.Node
		err := rows.Scan(
			&node.ID,
			&node.OldID,
			&node.AnyDeskAddress,
			&node.AnyDeskPassword,
			&node.Status,
			&node.Software,
			&node.Price,
			&node.Renter,
			&node.RentStartTime,
			&node.LastBalanceUpdateTimestamp,
			&node.CPU,
			&node.GPU,
			&node.OtherSpecs,
			&node.Licenses,
			&node.MachineID,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nodes = append(nodes, node)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

func UpdateNode(w http.ResponseWriter, r *http.Request) {
	var node models.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE nodes SET 
		status=$1, 
		software=$2, 
		price=$3, 
		renter=$4, 
		rent_start_time=$5, 
		last_balance_update_timestamp=$6, 
		cpu=$7, 
		gpu=$8, 
		other_specs=$9, 
		licenses=$10, 
		machine_id=$11
		WHERE any_desk_address=$12
	`
	_, err := db.PostgresEngine.Exec(
		query,
		node.Status,
		node.Software,
		node.Price,
		node.Renter,
		node.RentStartTime,
		node.LastBalanceUpdateTimestamp,
		node.CPU,
		node.GPU,
		node.OtherSpecs,
		node.Licenses,
		node.MachineID,
		node.AnyDeskAddress,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
