package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
	"strings"
	"time"
)

func GetRentSessions(w http.ResponseWriter, r *http.Request) {
	// Possible filters
	renter := r.URL.Query().Get("renter")
	status := r.URL.Query().Get("status")
	platform := r.URL.Query().Get("platform")
	nodeID := r.URL.Query().Get("node_id") // OPTIONAL new filter

	query := `
		SELECT
			id,
			renter,
			node_id,
			status,
			platform,
			total_price,
			rent_start_time,
			last_balance_update_timestamp,
			rent_stop_time
		FROM rent_sessions
		WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

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
	if platform != "" {
		query += fmt.Sprintf(" AND platform = $%d", argIndex)
		args = append(args, platform)
		argIndex++
	}
	if nodeID != "" {
		query += fmt.Sprintf(" AND node_id = $%d", argIndex)
		args = append(args, nodeID)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "GetRentSessions postgres query error",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var sessions []models.RentSession
	for rows.Next() {
		var s models.RentSession
		err := rows.Scan(
			&s.ID,
			&s.Renter,
			&s.NodeID,
			&s.Status,
			&s.Platform,
			&s.TotalPrice,
			&s.RentStartTime,
			&s.LastBalanceUpdateTimestamp,
			&s.RentStopTime,
		)
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "GetRentSessions scanning error",
				Error:   err.Error(),
			})
			return
		}
		sessions = append(sessions, s)
	}

	if len(sessions) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "No rent sessions found matching the criteria",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d rent session(s)", len(sessions)),
		Data:    sessions,
	})
}

func CreateRentSession(w http.ResponseWriter, r *http.Request) {
	var input models.RentSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "CreateRentSession JSON decoding error: " + err.Error(),
		})
		return
	}

	// Basic validations (expand as needed)
	if input.Renter == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "renter is required",
		})
		return
	}
	if input.Status == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "status is required",
		})
		return
	}
	if input.Platform == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "platform is required",
		})
		return
	}

	// Convert string timestamps to time.Time if needed
	var rentStartTime *time.Time
	var lastBalanceUpdate *time.Time
	var rentStopTime *time.Time

	if input.RentStartTime != nil && *input.RentStartTime != "" {
		t, e := time.Parse(time.RFC3339, *input.RentStartTime)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid rent_start_time format, must be RFC3339",
			})
			return
		}
		rentStartTime = &t
	}
	if input.LastBalanceUpdateTimestamp != nil && *input.LastBalanceUpdateTimestamp != "" {
		t, e := time.Parse(time.RFC3339, *input.LastBalanceUpdateTimestamp)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid last_balance_update_timestamp format, must be RFC3339",
			})
			return
		}
		lastBalanceUpdate = &t
	}
	if input.RentStopTime != nil && *input.RentStopTime != "" {
		t, e := time.Parse(time.RFC3339, *input.RentStopTime)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid rent_stop_time format, must be RFC3339",
			})
			return
		}
		rentStopTime = &t
	}

	// Build the INSERT query fields/values
	queryFields := []string{"renter", "status", "platform", "total_price"}
	queryValues := []string{"$1", "$2", "$3", "$4"}
	args := []interface{}{input.Renter, input.Status, input.Platform, input.TotalPrice}
	argIndex := 5

	// node_id is optional, but if provided, we insert
	if input.NodeID != nil {
		queryFields = append(queryFields, "node_id")
		queryValues = append(queryValues, fmt.Sprintf("$%d", argIndex))
		args = append(args, *input.NodeID)
		argIndex++
	}

	if rentStartTime != nil {
		queryFields = append(queryFields, "rent_start_time")
		queryValues = append(queryValues, fmt.Sprintf("$%d", argIndex))
		args = append(args, *rentStartTime)
		argIndex++
	}

	if lastBalanceUpdate != nil {
		queryFields = append(queryFields, "last_balance_update_timestamp")
		queryValues = append(queryValues, fmt.Sprintf("$%d", argIndex))
		args = append(args, *lastBalanceUpdate)
		argIndex++
	}

	if rentStopTime != nil {
		queryFields = append(queryFields, "rent_stop_time")
		queryValues = append(queryValues, fmt.Sprintf("$%d", argIndex))
		args = append(args, *rentStopTime)
		argIndex++
	}

	insertQuery := fmt.Sprintf(`
		INSERT INTO rent_sessions (%s)
		VALUES (%s)
		RETURNING
			id,
			renter,
			node_id,
			status,
			platform,
			total_price,
			rent_start_time,
			last_balance_update_timestamp,
			rent_stop_time
	`, strings.Join(queryFields, ", "), strings.Join(queryValues, ", "))

	var newSession models.RentSession
	err := db.PostgresEngine.QueryRow(insertQuery, args...).Scan(
		&newSession.ID,
		&newSession.Renter,
		&newSession.NodeID,
		&newSession.Status,
		&newSession.Platform,
		&newSession.TotalPrice,
		&newSession.RentStartTime,
		&newSession.LastBalanceUpdateTimestamp,
		&newSession.RentStopTime,
	)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "CreateRentSession postgres query error",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Rent session created successfully",
		Data:    newSession,
	})
}

func UpdateRentSession(w http.ResponseWriter, r *http.Request) {
	var input models.RentSessionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "UpdateRentSession JSON decoding error: " + err.Error(),
		})
		return
	}

	// ID is mandatory for updates
	if input.ID == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "rent_session ID is required for update",
		})
		return
	}

	// We build a dynamic UPDATE query
	query := `
		UPDATE rent_sessions
		SET
	`
	var sets []string
	var args []interface{}
	argIndex := 1

	if input.Renter != 0 {
		sets = append(sets, fmt.Sprintf("renter = $%d", argIndex))
		args = append(args, input.Renter)
		argIndex++
	}
	if input.NodeID != nil {
		// If node_id is explicitly provided (even if it's 0)
		sets = append(sets, fmt.Sprintf("node_id = $%d", argIndex))
		args = append(args, *input.NodeID)
		argIndex++
	}
	if input.Status != "" {
		sets = append(sets, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, input.Status)
		argIndex++
	}
	if input.Platform != "" {
		sets = append(sets, fmt.Sprintf("platform = $%d", argIndex))
		args = append(args, input.Platform)
		argIndex++
	}
	if input.TotalPrice != 0 {
		sets = append(sets, fmt.Sprintf("total_price = $%d", argIndex))
		args = append(args, input.TotalPrice)
		argIndex++
	}

	// Parse any times
	if input.RentStartTime != nil && *input.RentStartTime != "" {
		t, e := time.Parse(time.RFC3339, *input.RentStartTime)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid rent_start_time format, must be RFC3339",
			})
			return
		}
		sets = append(sets, fmt.Sprintf("rent_start_time = $%d", argIndex))
		args = append(args, t)
		argIndex++
	}
	if input.LastBalanceUpdateTimestamp != nil && *input.LastBalanceUpdateTimestamp != "" {
		t, e := time.Parse(time.RFC3339, *input.LastBalanceUpdateTimestamp)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid last_balance_update_timestamp format, must be RFC3339",
			})
			return
		}
		sets = append(sets, fmt.Sprintf("last_balance_update_timestamp = $%d", argIndex))
		args = append(args, t)
		argIndex++
	}
	if input.RentStopTime != nil && *input.RentStopTime != "" {
		t, e := time.Parse(time.RFC3339, *input.RentStopTime)
		if e != nil {
			utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
				Success: false,
				Error:   "Invalid rent_stop_time format, must be RFC3339",
			})
			return
		}
		sets = append(sets, fmt.Sprintf("rent_stop_time = $%d", argIndex))
		args = append(args, t)
		argIndex++
	}

	if len(sets) == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "No fields to update",
		})
		return
	}

	// Finalize the query and add the WHERE clause
	query += strings.Join(sets, ", ")
	query += fmt.Sprintf(" WHERE id = $%d", argIndex)
	args = append(args, input.ID)
	argIndex++

	// Return the updated row
	query += `
		RETURNING
			id,
			renter,
			node_id,
			status,
			platform,
			total_price,
			rent_start_time,
			last_balance_update_timestamp,
			rent_stop_time
	`

	var updatedSession models.RentSession
	err := db.PostgresEngine.QueryRow(query, args...).Scan(
		&updatedSession.ID,
		&updatedSession.Renter,
		&updatedSession.NodeID,
		&updatedSession.Status,
		&updatedSession.Platform,
		&updatedSession.TotalPrice,
		&updatedSession.RentStartTime,
		&updatedSession.LastBalanceUpdateTimestamp,
		&updatedSession.RentStopTime,
	)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "UpdateRentSession postgres query error",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Rent session updated successfully",
		Data:    updatedSession,
	})
}
