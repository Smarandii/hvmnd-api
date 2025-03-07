package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
)

func GetNotifications(w http.ResponseWriter, r *http.Request) {
	user_id := r.URL.Query().Get("user_id")
	is_read := r.URL.Query().Get("is_read")
	is_sent := r.URL.Query().Get("is_sent")
	notification_platform := r.URL.Query().Get("notification_platform")

	query := `
		SELECT
		id,
		user_id,
		notification_text,
		notification_platform,
		created_at,
		updated_at,
		is_read,
		is_sent
		FROM notifications WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if user_id != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, user_id)
		argIndex++
	}
	if is_read != "" {
		query += fmt.Sprintf(" AND is_read = $%d", argIndex)
		args = append(args, is_read)
		argIndex++
	}
	if is_sent != "" {
		query += fmt.Sprintf(" AND is_sent = $%d", argIndex)
		args = append(args, is_sent)
		argIndex++
	}
	if notification_platform != "" {
		query += fmt.Sprintf(" AND notification_platform = $%d", argIndex)
		args = append(args, notification_platform)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "GetNotifications postgres query error.",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var notification models.Notification
		err := rows.Scan(
			&notification.ID,
			&notification.UserId,
			&notification.NotificationText,
			&notification.NotificationPlatform,
			&notification.CreatedAt,
			&notification.UpdatedAt,
			&notification.IsRead,
			&notification.IsSent,
		)
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "GetNotifications postgres query error.",
				Error:   err.Error(),
			})
			return
		}
		notifications = append(notifications, notification)
	}

	if len(notifications) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "No notifications found matching the criteria",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d notifications", len(notifications)),
		Data:    notifications,
	})
}

func CreateNotification(w http.ResponseWriter, r *http.Request) {
	var input models.NotificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "CreateNotification JSON decoding error: " + err.Error(),
		})
		return
	}

	if input.UserId == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "user_id is required",
		})
		return
	}

	if input.NotificationText == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "notification_text is required",
		})
		return
	}

	if input.NotificationPlatform == "" {
		input.NotificationPlatform = "all"
	}

	query := `
		INSERT INTO notifications (
			user_id,
			notification_text,
			notification_platform
		)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, notification_text, notification_platform, created_at, updated_at, is_read, is_sent
	`

	var notification models.Notification
	err := db.PostgresEngine.QueryRow(
		query,
		input.UserId,
		input.NotificationText,
		input.NotificationPlatform,
	).Scan(
		&notification.ID,
		&notification.UserId,
		&notification.NotificationText,
		&notification.NotificationPlatform,
		&notification.CreatedAt,
		&notification.UpdatedAt,
		&notification.IsRead,
		&notification.IsSent,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "CreateNotification postgres query error.",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notification created successfully",
		Data:    notification,
	})
}

func UpdateNotification(w http.ResponseWriter, r *http.Request) {
	var input models.NotificationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "UpdateNotification JSON decoding error: " + err.Error(),
		})
		return
	}

	if input.ID == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "notification ID is required",
		})
		return
	}

	// Start building the SQL query
	query := `
		UPDATE notifications
		SET updated_at = CURRENT_TIMESTAMP
	`
	var args []interface{}
	argIndex := 1

	// Dynamically add fields based on input
	if input.UserId != 0 {
		query += fmt.Sprintf(", user_id = $%d", argIndex)
		args = append(args, input.UserId)
		argIndex++
	}
	if input.NotificationText != "" {
		query += fmt.Sprintf(", notification_text = $%d", argIndex)
		args = append(args, input.NotificationText)
		argIndex++
	}
	if input.NotificationPlatform != "" {
		query += fmt.Sprintf(", notification_platform = $%d", argIndex)
		args = append(args, input.NotificationPlatform)
		argIndex++
	}
	if input.IsRead {
		query += fmt.Sprintf(", is_read = $%d", argIndex)
		args = append(args, input.IsRead)
		argIndex++
	}
	if input.IsSent {
		query += fmt.Sprintf(", is_sent = $%d", argIndex)
		args = append(args, input.IsSent)
		argIndex++
	}

	// Add WHERE clause
	query += fmt.Sprintf(" WHERE id = $%d RETURNING id, user_id, notification_text, notification_platform, created_at, updated_at, is_read, is_sent", argIndex)
	args = append(args, input.ID)

	var notification models.Notification
	err := db.PostgresEngine.QueryRow(query, args...).Scan(
		&notification.ID,
		&notification.UserId,
		&notification.NotificationText,
		&notification.NotificationPlatform,
		&notification.CreatedAt,
		&notification.UpdatedAt,
		&notification.IsRead,
		&notification.IsSent,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "UpdateNotification postgres query error.",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Notification updated successfully",
		Data:    notification,
	})
}
