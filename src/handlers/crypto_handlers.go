package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
)

// GetUserDepositAddresses retrieves deposit addresses for the authenticated user
// If network_id is provided, returns a specific network address
// Otherwise returns all addresses
func GetUserDepositAddresses(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "user_id query parameter is required",
		})
		return
	}
	networkID := r.URL.Query().Get("network_id")
	if networkID != "" {
		// Get specific network address
		var addr models.CryptoDepositAddress
		var network models.CryptoNetwork

		err := db.PostgresEngine.QueryRow(`
			SELECT da.id, da.user_id, da.network_id, da.address, da.is_used, da.created_at, da.used_at,
				n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
			FROM crypto_deposit_addresses da
			JOIN crypto_networks n ON da.network_id = n.id
			WHERE da.user_id = $1 AND da.network_id = $2
		`, userID, networkID).Scan(
			&addr.ID, &addr.UserID, &addr.NetworkID, &addr.Address, &addr.IsUsed, &addr.CreatedAt, &addr.UsedAt,
			&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
		)

		if err == sql.ErrNoRows {
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Message: "No deposit address found for this network",
			})
			return
		} else if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to retrieve deposit address",
				Error:   err.Error(),
			})
			return
		}

		addr.Network = &network

		utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
			Success: true,
			Message: "Deposit address retrieved successfully",
			Data:    addr,
		})
		return
	}

	// Otherwise, get all addresses
	// Query all deposit addresses for this user
	rows, err := db.PostgresEngine.Query(`
		SELECT da.id, da.user_id, da.network_id, da.address, da.is_used, da.created_at, da.used_at,
		       n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
		FROM crypto_deposit_addresses da
		JOIN crypto_networks n ON da.network_id = n.id
		WHERE da.user_id = $1
	`, userID)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve deposit addresses",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var addresses []models.CryptoDepositAddress
	for rows.Next() {
		var addr models.CryptoDepositAddress
		var network models.CryptoNetwork

		err := rows.Scan(
			&addr.ID, &addr.UserID, &addr.NetworkID, &addr.Address, &addr.IsUsed, &addr.CreatedAt, &addr.UsedAt,
			&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
		)
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Error scanning deposit address data",
				Error:   err.Error(),
			})
			return
		}

		addr.Network = &network
		addresses = append(addresses, addr)
	}

	if len(addresses) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "No deposit addresses found for this user",
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Deposit addresses retrieved successfully",
		Data:    addresses,
	})
}

// CreateDepositAddress generates a new deposit address for the user if one doesn't exist
func CreateDepositAddress(w http.ResponseWriter, r *http.Request) {
	// Get network ID from request body
	var input struct {
		UserID    int `json:"user_id"`
		NetworkID int `json:"network_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request body",
			Error:   err.Error(),
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

	// Check if address already exists for this user and network
	var addressExists bool
	err = db.PostgresEngine.QueryRow("SELECT EXISTS(SELECT 1 FROM crypto_deposit_addresses WHERE user_id = $1 AND network_id = $2)",
		input.UserID, input.NetworkID).Scan(&addressExists)
	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to check existing address",
			Error:   err.Error(),
		})
		return
	}

	if addressExists {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Deposit address already exists for this network",
		})
		return
	}

	// Generate new address based on network type
	var depositAddress string
	var encryptedPrivateKey string

	// Currently only supporting TRC-20
	if input.NetworkID == 1 { // Assuming 1 is TRC-20
		var err error
		depositAddress, encryptedPrivateKey, err = utils.GenerateTRC20DepositWalletAddress()
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to generate deposit address",
				Error:   err.Error(),
			})
			return
		}
	} else {
		utils.WriteJSONResponse(w, http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Unsupported network for address generation",
		})
		return
	}

	// Insert the new address
	var newAddressID int
	err = db.PostgresEngine.QueryRow(`
        INSERT INTO crypto_deposit_addresses
        (user_id, network_id, address, encrypted_private_key, is_used, created_at)
        VALUES ($1, $2, $3, $4, false, NOW())
		RETURNING id`, input.UserID, input.NetworkID, depositAddress, encryptedPrivateKey,
	).Scan(&newAddressID)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to save deposit address",
			Error:   err.Error(),
		})
		return
	}

	// Retrieve the full address record with network info
	var addr models.CryptoDepositAddress
	var network models.CryptoNetwork

	err = db.PostgresEngine.QueryRow(`
		SELECT da.id, da.user_id, da.network_id, da.address, da.is_used, da.created_at, da.used_at,
		       n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
		FROM crypto_deposit_addresses da
		JOIN crypto_networks n ON da.network_id = n.id
		WHERE da.id = $1
	`, newAddressID).Scan(
		&addr.ID, &addr.UserID, &addr.NetworkID, &addr.Address, &addr.IsUsed, &addr.CreatedAt, &addr.UsedAt,
		&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
	)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve created address",
			Error:   err.Error(),
		})
		return
	}

	addr.Network = &network

	utils.WriteJSONResponse(w, http.StatusCreated, models.APIResponse{
		Success: true,
		Message: "Deposit address created successfully",
		Data:    addr,
	})
}
