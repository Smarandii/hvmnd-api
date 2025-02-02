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
	sessionToken := r.URL.Query().Get("session_token")
	id := r.URL.Query().Get("id")
	email := r.URL.Query().Get("email")

	query := `
        SELECT 
            u.id, 
            u.email,
            u.balance,
            u.total_spent,
            u.banned,
			u.email_confirmed,
			u.confirmation_token
        FROM webapp_users u
        WHERE 1=1
    `
	var args []interface{}
	argIndex := 1

	if email != "" {
		query += fmt.Sprintf(" AND u.email = $%d", argIndex)
		args = append(args, email)
		argIndex++
	}

	if id != "" {
		query += fmt.Sprintf(" AND u.id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}

	if sessionToken != "" {
		// If a session_token is provided, find the user by that token
		// Ensure the token is valid (expires_at > NOW())
		query += fmt.Sprintf(` AND u.id = (
            SELECT user_id FROM webapp_session_tokens 
            WHERE value = $%d AND expires_at > NOW()
            LIMIT 1
        )`, argIndex)
		args = append(args, sessionToken)
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
			&user.EmailConfirmed,
			&user.EmailConfirmationToken,
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
		INSERT INTO public.webapp_users (email, password_hash)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, balance, total_spent, banned, email_confirmed, confirmation_token`,
		input.Email,
		hashedPassword,
	).Scan(
		&newUser.ID,
		&newUser.Email,
		&newUser.Balance,
		&newUser.TotalSpent,
		&newUser.Banned,
		&newUser.EmailConfirmed,
		&newUser.EmailConfirmationToken,
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

func ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the confirmation token from the query parameters
	token := r.URL.Query().Get("token")
	if token == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Missing confirmation token in request",
		})
		return
	}

	// Update the user's email confirmation status
	result, err := db.PostgresEngine.Exec(`
		UPDATE public.webapp_users
		SET email_confirmed = TRUE,
			balance = balance + 3.00
		WHERE confirmation_token = $1 AND email_confirmed = FALSE
	`, token)

	if err != nil {
		// Database error while trying to confirm email
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to confirm email",
			Error:   err.Error(),
		})
		return
	}

	// Check how many rows were affected
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// No rows were updated, meaning invalid, expired token, or already confirmed email
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid, expired, or already confirmed confirmation token",
		})
		return
	}

	// Email successfully confirmed and bonus applied
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Email successfully confirmed! A $3 bonus has been added to your balance.",
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

func UpdateWebAppUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.WriteJSONResponse(w, http.StatusMethodNotAllowed, models.APIResponse{
			Success: false,
			Message: "Method not allowed, please use PATCH",
		})
		return
	}

	var input struct {
		ID         int     `json:"id"`
		Balance    float64 `json:"balance"`
		TotalSpent float64 `json:"total_spent"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// Ensure the user ID is provided.
	if input.ID == 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Missing 'id' field for the user to update",
		})
		return
	}

	// Prepare a statement updating only balance and total_spent.
	var updatedUser models.WebAppUser
	err := db.PostgresEngine.QueryRow(`
        UPDATE public.webapp_users
        SET 
            balance = $2,
            total_spent = $3
        WHERE id = $1
        RETURNING id, email, balance, total_spent, banned, email_confirmed, confirmation_token
    `,
		input.ID,
		input.Balance,
		input.TotalSpent,
	).Scan(
		&updatedUser.ID,
		&updatedUser.Email,
		&updatedUser.Balance,
		&updatedUser.TotalSpent,
		&updatedUser.Banned,
		&updatedUser.EmailConfirmed,
		&updatedUser.EmailConfirmationToken,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// No user with that ID
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Message: fmt.Sprintf("User with id=%d not found", input.ID),
			})
			return
		}
		// Some other DB error
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to update user in the database",
			Error:   err.Error(),
		})
		return
	}

	// Success!
	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Web app user updated successfully",
		Data:    updatedUser,
	})
}
