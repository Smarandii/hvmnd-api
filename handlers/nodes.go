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
	anyDeskAddress := r.URL.Query().Get("any_desk_address")
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
		if renter == "non_null" {
			query += " AND renter IS NOT NULL"
		} else {
			query += fmt.Sprintf(" AND renter = $%d", argIndex)
			args = append(args, renter)
			argIndex++
		}
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
		// Use two placeholders: one for software and one for licenses
		query += fmt.Sprintf(" AND (software ILIKE $%d OR licenses ILIKE $%d)", argIndex, argIndex+1)
		args = append(args, "%"+software+"%", "%"+software+"%")

		// Increment argIndex by 2 because we've used two placeholders
		argIndex += 2
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

	if nodes == nil {
		writeJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "No nodes found matching the criteria",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d nodes", len(nodes)),
		Data:    nodes,
	})
}

func UpdateNode(w http.ResponseWriter, r *http.Request) {
	var node models.NodeInput
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Ensure that at least one identifier is provided
	if node.AnyDeskAddress == nil && node.OldID == nil && node.ID == nil {
		http.Error(w, "At least one of any_desk_address, old_id, or id must be provided", http.StatusBadRequest)
		return
	}

	// Start with the base update query
	query := `
		UPDATE nodes SET 
			status = COALESCE($1, nodes.status),
			software = COALESCE($2, nodes.software),
			price = COALESCE($3, nodes.price),
			renter = COALESCE($4, nodes.renter),
			rent_start_time = COALESCE($5, nodes.rent_start_time),
			last_balance_update_timestamp = COALESCE($6, nodes.last_balance_update_timestamp),
			cpu = COALESCE($7, nodes.cpu),
			gpu = COALESCE($8, nodes.gpu),
			other_specs = COALESCE($9, nodes.other_specs),
			licenses = COALESCE($10, nodes.licenses),
			machine_id = COALESCE($11, nodes.machine_id),
			old_id = COALESCE($12, nodes.old_id),
			any_desk_address = COALESCE($13, nodes.any_desk_address)
		WHERE `

	// Arguments to pass into the query for the node's fields
	args := []interface{}{
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
		node.OldID,
		node.AnyDeskAddress,
	}

	// Dynamically add the WHERE clause based on the unique key provided
	if node.ID != nil {
		query += "id = $14"
		args = append(args, *node.ID)
	} else if node.OldID != nil {
		query += "old_id = $14"
		args = append(args, *node.OldID)
	} else if node.AnyDeskAddress != nil {
		query += "any_desk_address = $14"
		args = append(args, *node.AnyDeskAddress)
	}

	// Execute the query
	result, err := db.PostgresEngine.Exec(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check how many rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If no rows were affected, return 404 Not Found
	if rowsAffected == 0 {
		writeJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "Node not found or no changes applied",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Node updated successfully",
	})
}
