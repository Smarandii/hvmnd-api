package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
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
    // Read the raw body first
    body, err := ioutil.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Decode into a map to check which fields are present and if they are null
    var inputMap map[string]interface{}
    if err := json.Unmarshal(body, &inputMap); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Decode into the node struct
    var node models.NodeInput
    if err := json.Unmarshal(body, &node); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Ensure that at least one identifier is provided
    if node.AnyDeskAddress == nil && node.OldID == nil && node.ID == nil {
        http.Error(w, "At least one of any_desk_address, old_id, or id must be provided", http.StatusBadRequest)
        return
    }

    // Start building the UPDATE query dynamically
    query := "UPDATE nodes SET "
    sets := []string{}
    args := []interface{}{}
    argIndex := 1

    // Helper function to handle fields
    setField := func(fieldName string, fieldValue interface{}, inputVal interface{}) {
        // Check presence in inputMap:
        val, present := inputMap[fieldName]
        if !present {
            // Field not provided at all, do not update this column
            return
        }

        // Field is provided
        if val == nil {
            // Explicitly set to NULL
            sets = append(sets, fmt.Sprintf("%s = NULL", fieldName))
        } else {
            // Set to given value (non-null)
            sets = append(sets, fmt.Sprintf("%s = $%d", fieldName, argIndex))
            args = append(args, fieldValue)
            argIndex++
        }
    }

    // Call setField for each updatable column
    // Here we rely on node.* fields and their presence in inputMap.
    // If a field is a pointer and node.* is nil, that means user passed null.
    // If the field is absent from inputMap, we don't update that field at all.

    setField("status", node.Status, inputMap["status"])
    setField("software", node.Software, inputMap["software"])
    setField("price", node.Price, inputMap["price"])
    setField("renter", node.Renter, inputMap["renter"])
    setField("rent_start_time", node.RentStartTime, inputMap["rent_start_time"])
    setField("last_balance_update_timestamp", node.LastBalanceUpdateTimestamp, inputMap["last_balance_update_timestamp"])
    setField("cpu", node.CPU, inputMap["cpu"])
    setField("gpu", node.GPU, inputMap["gpu"])
    setField("other_specs", node.OtherSpecs, inputMap["other_specs"])
    setField("licenses", node.Licenses, inputMap["licenses"])
    setField("machine_id", node.MachineID, inputMap["machine_id"])
    setField("old_id", node.OldID, inputMap["old_id"])
    setField("any_desk_address", node.AnyDeskAddress, inputMap["any_desk_address"])

	if len(sets) == 0 {
        writeJSONResponse(w, http.StatusNotFound, APIResponse{
            Success: false,
            Error:   "No updatable fields provided",
        })
        return
    }

    query += strings.Join(sets, ", ") + " WHERE "

    // Dynamically add the WHERE clause based on the unique key provided
    if node.ID != nil {
        query += fmt.Sprintf("id = $%d", argIndex)
        args = append(args, *node.ID)
    } else if node.OldID != nil {
        query += fmt.Sprintf("old_id = $%d", argIndex)
        args = append(args, *node.OldID)
    } else if node.AnyDeskAddress != nil {
        query += fmt.Sprintf("any_desk_address = $%d", argIndex)
        args = append(args, *node.AnyDeskAddress)
    }
    // Now argIndex not incremented here because we only added one condition.

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

