package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"io"
	"log"
	"net/http"
	"strings"
)

func GetNodes(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}

	status := r.URL.Query().Get("status")
	anyDeskAddress := r.URL.Query().Get("any_desk_address")
	machineId := r.URL.Query().Get("machine_id")
	software := r.URL.Query().Get("software")

	query := `
		SELECT 
		id, old_id, any_desk_address, 
		any_desk_password, status, software, 
		price, cpu, gpu, other_specs, licenses, 
		machine_id FROM nodes WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
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

	if machineId != "" {
		query += fmt.Sprintf(" AND machine_id = $%d", argIndex)
		args = append(args, machineId)
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
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "GetNodes postgres query error.",
			Error:   err.Error(),
		})
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
			&node.CPU,
			&node.GPU,
			&node.OtherSpecs,
			&node.Licenses,
			&node.MachineID,
		)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "GetNodes rows scan error.",
				Error:   err.Error(),
			})
			return
		}

		nodes = append(nodes, node)
	}

	if nodes == nil {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "No nodes found matching the criteria",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d nodes", len(nodes)),
		Data:    nodes,
	})
}

func CreateNode(w http.ResponseWriter, r *http.Request) {
	// Read the raw body first
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "CreateNode read body error.",
			Error:   err.Error(),
		})
		return
	}

	// Decode into the node struct
	var node models.NodeInput
	if err := json.Unmarshal(body, &node); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Ensure that the required fields are provided
	if node.AnyDeskAddress == nil ||
		node.AnyDeskPassword == nil ||
		node.Status == nil ||
		node.MachineID == nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Required fields missing: any_desk_address, status, and price must be provided",
		})
		return
	}

	// Start building the INSERT query dynamically
	query := "INSERT INTO nodes (any_desk_address, any_desk_password, status, machine_id) VALUES ("
	values := []interface{}{}
	argIndex := 1

	// Helper function to handle fields dynamically
	setField := func(fieldValue interface{}) {
		if argIndex > 1 {
			query += ","
		}
		if fieldValue != nil {
			query += fmt.Sprintf("$%d", argIndex)
			values = append(values, fieldValue)
			argIndex++
		} else {
			query += "NULL"
		}
	}

	// Add values dynamically for each field
	setField(node.AnyDeskAddress)
	setField(node.AnyDeskPassword)
	setField(node.Status)
	setField(node.MachineID)

	query += ") RETURNING id"

	// Execute the query
	var nodeID int
	err = db.PostgresEngine.QueryRow(query, values...).Scan(&nodeID)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "CreateNode postgres query error.",
			Error:   err.Error(),
		})
		return
	}

	// Return the created node's ID
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Node created successfully with ID %d", nodeID),
		Data: map[string]int{
			"id": nodeID,
		},
	})
}

func UpdateNode(w http.ResponseWriter, r *http.Request) {
	// Read the raw body first
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "UpdateNode read body error.",
			Error:   err.Error(),
		})
		return
	}

	// Decode into a map to check which fields are present and if they are null
	var inputMap map[string]interface{}
	if err := json.Unmarshal(body, &inputMap); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Decode into the node struct
	var node models.NodeInput
	if err := json.Unmarshal(body, &node); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	// Ensure that at least one identifier is provided
	if node.AnyDeskAddress == nil && node.OldID == nil && node.ID == nil && node.MachineID == nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "At least one of any_desk_address, machine_id, old_id, or id must be provided",
		})
		return
	}

	// Start building the UPDATE query dynamically
	query := "UPDATE nodes SET "
	sets := []string{}
	args := []interface{}{}
	argIndex := 1

	// Helper function to handle fields
	setField := func(fieldName string, fieldValue interface{}) {
		// Check presence in inputMap:
		val, present := inputMap[fieldName]
		if !present {
			// Field not provided at all, do not update this column
			return
		}

		// Special case: do not allow machine_id to be set to NULL.
		if fieldName == "machine_id" && val == nil {
			// Instead of setting machine_id = NULL, skip updating it.
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

	setField("status", node.Status)
	setField("software", node.Software)
	setField("price", node.Price)
	setField("cpu", node.CPU)
	setField("gpu", node.GPU)
	setField("other_specs", node.OtherSpecs)
	setField("licenses", node.Licenses)
	setField("machine_id", node.MachineID)
	setField("old_id", node.OldID)
	setField("any_desk_address", node.AnyDeskAddress)
	setField("any_desk_password", node.AnyDeskPassword)

	if len(sets) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
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
	} else if node.MachineID != nil {
		query += fmt.Sprintf("machine_id = $%d", argIndex)
		args = append(args, *node.MachineID)
	} else if node.OldID != nil {
		query += fmt.Sprintf("old_id = $%d", argIndex)
		args = append(args, *node.OldID)
	} else if node.AnyDeskAddress != nil {
		query += fmt.Sprintf("any_desk_address = $%d", argIndex)
		args = append(args, *node.AnyDeskAddress)
	}
	// Now argIndex not incremented here because we only added one condition.

	log.Printf("UpdateNode postgres query: %q", query)
	// Execute the query
	result, err := db.PostgresEngine.Exec(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "UpdateNode postgres query error.",
			Error:   err.Error(),
		})
		return
	}

	// Check how many rows were affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "UpdateNode error while checking how many affected rows.",
			Error:   err.Error(),
		})
		return
	}

	// If no rows were affected, return 404 Not Found
	if rowsAffected == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "Node not found or no changes applied",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Node updated successfully",
	})
}
