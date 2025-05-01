package handlers

import (
	"encoding/json"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
)

// POST /api/v1/telegram/user-interactions
func CreateTgUserInteraction(w http.ResponseWriter, r *http.Request) {
	var input models.TgUserInteractionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid JSON body",
			Error:   err.Error(),
		})
		return
	}

	// Basic validation
	if input.TelegramID == 0 || input.EventType == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "`telegram_id` and `event_type` are required",
		})
		return
	}

	var created models.TgUserInteraction
	err := db.PostgresEngine.QueryRow(`
		INSERT INTO tg_user_interactions (telegram_id, event_type, event_data)
		VALUES ($1, $2, $3)
		RETURNING id, telegram_id, event_type, event_data, timestamp
	`, input.TelegramID, input.EventType, input.EventData).
		Scan(&created.ID, &created.TelegramID, &created.EventType, &created.EventData, &created.Timestamp)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to save interaction",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Interaction saved",
		Data:    created,
	})
}
