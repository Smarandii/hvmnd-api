package handlers

import (
	"database/sql"
	"net/http"

	"hvmnd/api/db"
	"hvmnd/api/models"
	"hvmnd/api/utils"
)

func GetUserCryptoTransactions(w http.ResponseWriter, r *http.Request) {
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
		// Get transactions for specific network
		rows, err := db.PostgresEngine.Query(`
			SELECT t.id, t.user_id, t.deposit_address_id, t.network_id, t.amount, 
				   t.transaction_hash, t.status, t.confirmations, 
				   t.created_at, t.confirmed_at,
				   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
			FROM crypto_payment_transactions t
			JOIN crypto_networks n ON t.network_id = n.id
			WHERE t.user_id = $1 AND t.network_id = $2
			ORDER BY t.created_at DESC
		`, userID, networkID)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to retrieve transactions",
				Error:   err.Error(),
			})
			return
		}
		defer rows.Close()

		var transactions []models.CryptoPaymentTransaction
		for rows.Next() {
			var tx models.CryptoPaymentTransaction
			var network models.CryptoNetwork
			var txHash sql.NullString
			var confirmedTime sql.NullTime

			err := rows.Scan(
				&tx.ID, &tx.UserID, &tx.DepositAddressID, &tx.NetworkID, &tx.Amount, &txHash, &tx.Status, &tx.Confirmations,
				&tx.CreatedAt, &confirmedTime,
				&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
			)

			if err != nil {
				utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
					Success: false,
					Message: "Error scanning transaction data",
					Error:   err.Error(),
				})
				return
			}

			if txHash.Valid {
				tx.TransactionHash = txHash.String
			}

			if confirmedTime.Valid {
				tx.ConfirmedAt = &confirmedTime.Time
			}

			tx.Network = &network
			transactions = append(transactions, tx)
		}

		if len(transactions) == 0 {
			utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
				Success: false,
				Message: "No transactions found for this user and network",
				Data:    transactions,
			})
			return
		}

		utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
			Success: true,
			Message: "Transactions retrieved successfully",
			Data:    transactions,
		})
		return
	}

	// Otherwise, get all transactions for this user
	rows, err := db.PostgresEngine.Query(`
		SELECT t.id, t.user_id, t.deposit_address_id, t.network_id, t.amount, 
			   t.transaction_hash, t.status, t.confirmations, 
			   t.created_at, t.confirmed_at,
			   n.id, n.name, n.token_symbol, n.contract_address, n.network_fee, n.created_at
		FROM crypto_payment_transactions t
		JOIN crypto_networks n ON t.network_id = n.id
		WHERE t.user_id = $1
		ORDER BY t.created_at DESC
	`, userID)

	if err != nil {
		utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to retrieve transactions",
			Error:   err.Error(),
		})
		return
	}
	defer rows.Close()

	var transactions []models.CryptoPaymentTransaction
	for rows.Next() {
		var tx models.CryptoPaymentTransaction
		var network models.CryptoNetwork
		var txHash sql.NullString
		var confirmedTime sql.NullTime

		err := rows.Scan(
			&tx.ID, &tx.UserID, &tx.DepositAddressID, &tx.NetworkID, &tx.Amount, &txHash, &tx.Status, &tx.Confirmations,
			&tx.CreatedAt, &confirmedTime,
			&network.ID, &network.Name, &network.TokenSymbol, &network.ContractAddress, &network.NetworkFee, &network.CreatedAt,
		)

		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Error scanning transaction data",
				Error:   err.Error(),
			})
			return
		}

		if txHash.Valid {
			tx.TransactionHash = txHash.String
		}

		if confirmedTime.Valid {
			tx.ConfirmedAt = &confirmedTime.Time
		}

		tx.Network = &network
		transactions = append(transactions, tx)
	}

	if len(transactions) == 0 {
		utils.WriteJSONResponse(w, http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "No transactions found for this user",
			Data:    transactions,
		})
		return
	}

	utils.WriteJSONResponse(w, http.StatusOK, models.APIResponse{
		Success: true,
		Message: "Transactions retrieved successfully",
		Data:    transactions,
	})
}
