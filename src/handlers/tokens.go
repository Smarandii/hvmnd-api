package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
)

func GetTokens(w http.ResponseWriter, r *http.Request) {

	telegramID := r.URL.Query().Get("telegram_id")
	user_id := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")

	query := `
		SELECT 
		id,
		token,
		user_id,
		telegram_id,
		status,
		created_at
		FROM tokens WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if telegramID != "" {
		query += fmt.Sprintf(" AND telegram_id = $%d", argIndex)
		args = append(args, telegramID)
		argIndex++
	}
	if user_id != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, user_id)
		argIndex++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "GetTokens postgres query error.",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var tokens []models.Token
	for rows.Next() {
		var token models.Token
		err := rows.Scan(
			&token.ID,
			&token.Token,
			&token.UserId,
			&token.TelegramID,
			&token.Status,
			&token.CreatedAt,
		)
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "GetTokens postgres query error.",
				Error:   err.Error(),
			})
			return
		}
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "No tokens found matching the criteria",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d tokens", len(tokens)),
		Data:    tokens,
	})
}

func CreateToken(w http.ResponseWriter, r *http.Request) {
	var input models.TokenInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "CreateToken json decoding error:" + err.Error(),
		})
		return
	}

	if input.UserId == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "CreateToken error: user_id is required",
		})
		return
	}

	if input.TelegramID == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Error:   "CreateToken error: telegram_id is required",
		})
		return
	}

	if input.Status == "" {
		input.Status = "pending"
	}

	query := `
		INSERT INTO public.tokens (
			telegram_id, 
			user_id, 
			status
		)
		VALUES ($1, $2, $3)
		RETURNING id, token, user_id, telegram_id, status, created_at
	`

	var token models.Token
	err := db.PostgresEngine.QueryRow(
		query,
		input.TelegramID,
		input.UserId,
		input.Status,
	).Scan(
		&token.ID,
		&token.Token,
		&token.UserId,
		&token.TelegramID,
		&token.Status,
		&token.CreatedAt,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "CreateToken postgres query error.",
			Error:   err.Error(),
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Token created successfully",
		Data:    token,
	})
}
