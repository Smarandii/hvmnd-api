package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func GetWebAppUsers(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}
	email := r.URL.Query().Get("email")

	query := `
		SELECT 
		id, 
		email,
		balance,
		total_spent,
		banned
		FROM webapp_users WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if email != "" {
		query += fmt.Sprintf(" AND email = $%d", argIndex)
		args = append(args, email)
		argIndex++
	}
	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "GetWebAppUsers postgres query error.",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var users []models.WebAppUser
	for rows.Next() {
		var user models.WebAppUser
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Balance,
			&user.TotalSpent,
			&user.Banned,
		)
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "GetWebAppUsers rows scan error.",
				Error:   err.Error(),
			})
			return
		}
		users = append(users, user)
	}

	if len(users) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Error:   "No users found matching the criteria",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d users", len(users)),
		Data:    users,
	})
}

func RegisterWebAppUser(w http.ResponseWriter, r *http.Request) {
	var input models.RegistrationInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" || input.ConfirmPassword == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Email, password, and confirm password are required",
		})
		return
	}
	if input.Password != input.ConfirmPassword {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Passwords do not match",
		})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "RegisterWebAppUser failed to hash password.",
			Error:   err.Error(),
		})
		return
	}

	// Check if user already exists
	var existingUserID int
	err = db.PostgresEngine.QueryRow(
		"SELECT id FROM public.webapp_users WHERE email = $1",
		input.Email,
	).Scan(&existingUserID)

	if err != nil && err != sql.ErrNoRows {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "RegisterWebAppUser failed to query database.",
			Error:   err.Error(),
		})
		return
	}
	if existingUserID > 0 {
		utils.WriteJSONResponse(w, http.StatusConflict, models.APIResponse{
			Success: false,
			Message: "User with this email already exists",
		})
		return
	}

	// Create new user
	var newUser models.WebAppUser
	err = db.PostgresEngine.QueryRow(`
		INSERT INTO public.webapp_users (email, password_hash, balance, total_spent, banned)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, balance, total_spent, banned`,
		input.Email,
		hashedPassword,
		1.0,   // Default balance in USD dollars
		0.0,   // Default total_spent
		false, // Not banned by default
	).Scan(
		&newUser.ID,
		&newUser.Email,
		&newUser.Balance,
		&newUser.TotalSpent,
		&newUser.Banned,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "RegisterWebAppUser failed to create user.",
			Error:   err.Error(),
		})
		return
	}

	// Respond with the new user's information
	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Web app user registered successfully",
		Data:    newUser,
	})
}

func LoginWebAppUser(w http.ResponseWriter, r *http.Request) {
	var input models.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Email and password are required",
		})
		return
	}

	// Fetch the stored hash from the DB
	var (
		userID     int
		storedHash []byte
	)
	err := db.PostgresEngine.QueryRow(
		"SELECT id, password_hash FROM public.webapp_users WHERE email = $1",
		input.Email,
	).Scan(&userID, &storedHash)

	if err == sql.ErrNoRows {
		// User not found
		utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "User does not exist or password is wrong",
		})
		return
	} else if err != nil {
		// DB error
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Database error",
			Error:   err.Error(),
		})
		return
	}

	// Compare the provided password with the stored hash
	if err := bcrypt.CompareHashAndPassword(storedHash, []byte(input.Password)); err != nil {
		// Passwords don't match
		utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "User does not exist or password is wrong",
		})
		return
	}

	// Password correct, now handle session token logic
	// Check if a valid token already exists for this user
	var existingTokenID int
	err = db.PostgresEngine.QueryRow(`
        SELECT id FROM public.webapp_session_tokens 
        WHERE user_id = $1 AND expires_at > NOW() 
        LIMIT 1`,
		userID,
	).Scan(&existingTokenID)

	var newToken models.WebAppSessionToken
	if err != nil && err != sql.ErrNoRows {
		// Some DB error
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to query existing token",
			Error:   err.Error(),
		})
		return
	}

	if existingTokenID > 0 {
		// There is already a valid token, retrieve it
		err = db.PostgresEngine.QueryRow(`
            SELECT id, user_id, value, expires_at 
            FROM public.webapp_session_tokens 
            WHERE id = $1`,
			existingTokenID,
		).Scan(&newToken.ID, &newToken.UserID, &newToken.Value, &newToken.ExpiresAt)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to retrieve existing token",
				Error:   err.Error(),
			})
			return
		}
	} else {
		// No valid token, create a new one
		tokenValue := utils.GenerateRandomToken(32) // Implement your own random token generator
		expiresAt := time.Now().Add(24 * time.Hour)

		err = db.PostgresEngine.QueryRow(`
            INSERT INTO public.webapp_session_tokens (user_id, value, expires_at)
            VALUES ($1, $2, $3)
            RETURNING id, user_id, value, expires_at
        `, userID, tokenValue, expiresAt).Scan(
			&newToken.ID, &newToken.UserID, &newToken.Value, &newToken.ExpiresAt,
		)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "LoginWebAppUser failed to create token.",
				Error:   err.Error(),
			})
			return
		}
	}

	// Respond with the token
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Web app user logged in successfully",
		Data:    newToken,
	})
}
