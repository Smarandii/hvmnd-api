package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
)

// GetUserTopupIntents retrieves all crypto topup intents for a user
func GetUserTopupIntents(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "user_id query parameter is required",
		})
		return
	}

	networkID := r.URL.Query().Get("network_id")
	var rows *sql.Rows
	var err error

	if networkID != "" {
		// Get intents for specific network
		rows, err = db.PostgresEngine.Query(`
			SELECT i.id, i.user_id, i.network_id, i.amount, i.status, 
				   i.transaction_hash, i.created_at, i.confirmed_at, i.expires_at,
				   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
			FROM crypto_topup_intents i
			JOIN crypto_networks n ON i.network_id = n.id
			WHERE i.user_id = $1 AND i.network_id = $2
			ORDER BY i.created_at DESC
		`, userID, networkID)
	} else {
		// Get all intents for this user
		rows, err = db.PostgresEngine.Query(`
			SELECT i.id, i.user_id, i.network_id, i.amount, i.status, 
				   i.transaction_hash, i.created_at, i.confirmed_at, i.expires_at,
				   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
			FROM crypto_topup_intents i
			JOIN crypto_networks n ON i.network_id = n.id
			WHERE i.user_id = $1
			ORDER BY i.created_at DESC
		`, userID)
	}

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve topup intents",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var intents []models.CryptoTopupIntent
	for rows.Next() {
		var intent models.CryptoTopupIntent
		var network models.CryptoNetwork
		var txHash sql.NullString
		var confirmedTime sql.NullTime
		var expiresTime sql.NullTime

		err := rows.Scan(
			&intent.ID, &intent.UserID, &intent.NetworkID, &intent.Amount, &intent.Status,
			&txHash, &intent.CreatedAt, &confirmedTime, &expiresTime,
			&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
		)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Error scanning topup intent data",
				Error:   err.Error(),
			})
			return
		}

		if txHash.Valid {
			intent.TransactionHash = txHash.String
		}

		if confirmedTime.Valid {
			intent.ConfirmedAt = &confirmedTime.Time
		}

		if expiresTime.Valid {
			intent.ExpiresAt = &expiresTime.Time
		}

		intent.Network = &network
		intents = append(intents, intent)
	}

	if len(intents) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "No topup intents found for this user",
			Data:    intents,
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Topup intents retrieved successfully",
		Data:    intents,
	})
}

// CreateTopupIntent creates a new crypto topup intent
func CreateTopupIntent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		UserID    int     `json:"user_id"`
		NetworkID int     `json:"network_id"`
		Amount    float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if input.UserID <= 0 || input.NetworkID <= 0 || input.Amount <= 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid input: user_id, network_id must be positive integers and amount must be positive",
		})
		return
	}

	// Check if network exists
	var networkExists bool
	err := db.PostgresEngine.QueryRow("SELECT EXISTS(SELECT 1 FROM crypto_networks WHERE id = $1)", input.NetworkID).Scan(&networkExists)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to check network existence",
			Error:   err.Error(),
		})
		return
	}

	if !networkExists {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid network ID",
		})
		return
	}

	// Check if user already has a pending topup intent
	var existingIntent bool
	err = db.PostgresEngine.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM crypto_topup_intents 
			WHERE user_id = $1 AND status = $2
		)
	`, input.UserID, models.TopupIntentStatusPending).Scan(&existingIntent)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to check existing intents",
			Error:   err.Error(),
		})
		return
	}

	if existingIntent {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "User already has a pending topup intent",
		})
		return
	}

	// Create the topup intent
	// Default expiration time is 1 hour from now (handled by DB default)
	var newIntentID int
	err = db.PostgresEngine.QueryRow(`
		INSERT INTO crypto_topup_intents
		(user_id, network_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		RETURNING id
	`, input.UserID, input.NetworkID, input.Amount, models.TopupIntentStatusPending).Scan(&newIntentID)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to create topup intent",
			Error:   err.Error(),
		})
		return
	}

	// Retrieve the full intent record with network info
	var intent models.CryptoTopupIntent
	var network models.CryptoNetwork
	var txHash sql.NullString
	var confirmedTime sql.NullTime
	var expiresTime time.Time

	err = db.PostgresEngine.QueryRow(`
		SELECT i.id, i.user_id, i.network_id, i.amount, i.status, 
			   i.transaction_hash, i.created_at, i.confirmed_at, i.expires_at,
			   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
		FROM crypto_topup_intents i
		JOIN crypto_networks n ON i.network_id = n.id
		WHERE i.id = $1
	`, newIntentID).Scan(
		&intent.ID, &intent.UserID, &intent.NetworkID, &intent.Amount, &intent.Status,
		&txHash, &intent.CreatedAt, &confirmedTime, &expiresTime,
		&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve created topup intent",
			Error:   err.Error(),
		})
		return
	}

	if txHash.Valid {
		intent.TransactionHash = txHash.String
	}

	if confirmedTime.Valid {
		intent.ConfirmedAt = &confirmedTime.Time
	}

	intent.ExpiresAt = &expiresTime
	intent.Network = &network

	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Topup intent created successfully",
		Data:    intent,
	})
}

// CancelTopupIntent cancels a crypto topup intent if it's not in confirmed or confirming state
func CancelTopupIntent(w http.ResponseWriter, r *http.Request) {

	intentID, err := strconv.Atoi(r.PathValue("id"))

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid intent id",
			Error:   err.Error(),
		})
		return
	}

	// Validate input
	if intentID <= 0 {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Intent ID must be a positive integer",
		})
		return
	}

	// Check if intent exists and its current status
	var currentStatus string
	err = db.PostgresEngine.QueryRow("SELECT status FROM crypto_topup_intents WHERE id = $1", intentID).Scan(&currentStatus)
	if err != nil {
		if err == sql.ErrNoRows {
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Message: "Topup intent not found",
			})
			return
		}
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve topup intent",
			Error:   err.Error(),
		})
		return
	}

	// Check if the intent can be cancelled
	if currentStatus == models.TopupIntentStatusConfirmed || currentStatus == models.TopupIntentStatusConfirming {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Cannot cancel intent that is already confirming or confirmed",
		})
		return
	}

	// Update the intent status to cancelled
	_, err = db.PostgresEngine.Exec("UPDATE crypto_topup_intents SET status = $1 WHERE id = $2",
		models.TopupIntentStatusCancelled, intentID)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to cancel topup intent",
			Error:   err.Error(),
		})
		return
	}

	// Retrieve the updated intent record with network info
	var intent models.CryptoTopupIntent
	var network models.CryptoNetwork
	var txHash sql.NullString
	var confirmedTime sql.NullTime
	var expiresTime sql.NullTime

	err = db.PostgresEngine.QueryRow(`
		SELECT i.id, i.user_id, i.network_id, i.amount, i.status, 
			   i.transaction_hash, i.created_at, i.confirmed_at, i.expires_at,
			   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
		FROM crypto_topup_intents i
		JOIN crypto_networks n ON i.network_id = n.id
		WHERE i.id = $1
	`, intentID).Scan(
		&intent.ID, &intent.UserID, &intent.NetworkID, &intent.Amount, &intent.Status,
		&txHash, &intent.CreatedAt, &confirmedTime, &expiresTime,
		&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve updated topup intent",
			Error:   err.Error(),
		})
		return
	}

	if txHash.Valid {
		intent.TransactionHash = txHash.String
	}

	if confirmedTime.Valid {
		intent.ConfirmedAt = &confirmedTime.Time
	}

	if expiresTime.Valid {
		intent.ExpiresAt = &expiresTime.Time
	}

	intent.Network = &network

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Topup intent cancelled successfully",
		Data:    intent,
	})
}
