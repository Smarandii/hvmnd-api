package handlers

import (
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"net/http"
)

func GetUsers(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}
	telegramID := r.URL.Query().Get("telegram_id")
	username := r.URL.Query().Get("username")
	limit := r.URL.Query().Get("limit")

	query := `
		SELECT 
		id, 
		telegram_id, 
		total_spent, 
		balance, 
		first_name, 
		last_name, 
		username, 
		language_code,
		banned
		FROM users WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if telegramID != "" {
		query += fmt.Sprintf(" AND telegram_id = $%d", argIndex)
		args = append(args, telegramID)
		argIndex++
	}
	if username != "" {
		query += fmt.Sprintf(" AND username = $%d", argIndex)
		args = append(args, username)
		argIndex++
	}
	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}
	if limit != "" {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.TelegramID,
			&user.TotalSpent,
			&user.Balance,
			&user.FirstName,
			&user.LastName,
			&user.Username,
			&user.LanguageCode,
			&user.Banned,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	if len(users) == 0 {
		writeJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "No users found matching the criteria",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d users", len(users)),
		Data:    users,
	})
}

func CreateOrUpdateUser(w http.ResponseWriter, r *http.Request) {
	var input models.UserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if input.TelegramID == 0 {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO public.users (
			telegram_id, 
			total_spent, 
			balance, 
			first_name, 
			last_name, 
			username, 
			language_code,
			banned
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (telegram_id) DO UPDATE
		SET 
			total_spent = COALESCE(EXCLUDED.total_spent, public.users.total_spent), 
			balance = COALESCE(EXCLUDED.balance, public.users.balance), 
			first_name = COALESCE(EXCLUDED.first_name, public.users.first_name), 
			last_name = COALESCE(EXCLUDED.last_name, public.users.last_name), 
			username = COALESCE(EXCLUDED.username, public.users.username), 
			language_code = COALESCE(EXCLUDED.language_code, public.users.language_code),
			banned = COALESCE(EXCLUDED.banned, public.users.banned)
		WHERE public.users.telegram_id = EXCLUDED.telegram_id
		RETURNING id, telegram_id, total_spent, balance, first_name, last_name, username, language_code, banned
	`

	var user models.User
	err := db.PostgresEngine.QueryRow(
		query,
		input.TelegramID,
		input.TotalSpent,
		input.Balance,
		input.FirstName,
		input.LastName,
		input.Username,
		input.LanguageCode,
		input.Banned,
	).Scan(
		&user.ID,
		&user.TelegramID,
		&user.TotalSpent,
		&user.Balance,
		&user.FirstName,
		&user.LastName,
		&user.Username,
		&user.LanguageCode,
		&user.Banned,
	)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "User created/updated successfully",
		Data:    user,
	})
}
