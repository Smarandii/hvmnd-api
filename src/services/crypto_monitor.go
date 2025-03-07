package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"hvmnd/api/db"
	"hvmnd/api/models"
)

const (
	// How often to check for new transactions
	pollingInterval = 2 * time.Minute

	// Number of confirmations required to consider a transaction confirmed
	requiredConfirmations = 6
)

// TronGridTransferResponse represents the response from TronGrid API for TRC-20 transfers
type TronGridTransferResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		TransactionID string `json:"transaction_id"`
		TokenInfo     struct {
			Symbol   string `json:"symbol"`
			Address  string `json:"address"`
			Decimals int    `json:"decimals"`
		} `json:"token_info"`
		From   string `json:"from"`
		To     string `json:"to"`
		Amount string `json:"value"`
		Block  int    `json:"block_timestamp"`
	} `json:"data"`
}

// StartCryptoMonitor initializes and starts the crypto deposit monitoring service
func StartCryptoMonitor(ctx context.Context) {
	log.Println("Starting crypto deposit monitoring service...")

	// Start the monitoring loop in a goroutine
	go func() {
		ticker := time.NewTicker(pollingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Println("Stopping crypto deposit monitoring service...")
				return
			case <-ticker.C:
				log.Println("Checking for new crypto deposits...")
				if err := checkForNewDeposits(); err != nil {
					log.Printf("Error checking for deposits: %v", err)
				}
				if err := updatePendingTransactions(); err != nil {
					log.Printf("Error updating pending transactions: %v", err)
				}
			}
		}
	}()
}

// checkForNewDeposits queries TronGrid for new deposits to our addresses
func checkForNewDeposits() error {
	// Get all active deposit addresses
	rows, err := db.PostgresEngine.Query(`
		SELECT da.id, da.user_id, da.network_id, da.address, n.contract_address, n.token_symbol
		FROM crypto_deposit_addresses da
		JOIN crypto_networks n ON da.network_id = n.id
		WHERE n.name = 'TRC-20' AND da.is_used = false
	`)
	if err != nil {
		return fmt.Errorf("failed to query deposit addresses: %w", err)
	}
	defer rows.Close()

	// Process each address
	for rows.Next() {
		var addressID, userID, networkID int
		var address, contractAddress, tokenSymbol string

		if err := rows.Scan(&addressID, &userID, &networkID, &address, &contractAddress, &tokenSymbol); err != nil {
			log.Printf("Error scanning address row: %v", err)
			continue
		}

		// Check for new transactions for this address
		if err := checkAddressTransactions(addressID, userID, networkID, address, contractAddress, tokenSymbol); err != nil {
			log.Printf("Error checking transactions for address %s: %v", address, err)
		}
	}

	return nil
}

// checkAddressTransactions queries TronGrid for transactions involving a specific address
func checkAddressTransactions(addressID, userID, networkID int, address, contractAddress, tokenSymbol string) error {
	// Get TronGrid API key from environment
	apiKey := os.Getenv("TRON_GRID_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("TRON_GRID_API_KEY not set")
	}

	// Construct TronGrid API URL for TRC-20 transfers
	url := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s/transactions/trc20?contract_address=%s&only_confirmed=true&limit=20",
		address, contractAddress)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Add("TRON-PRO-API-KEY", apiKey)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var tronResp TronGridTransferResponse
	if err := json.NewDecoder(resp.Body).Decode(&tronResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !tronResp.Success {
		return fmt.Errorf("TronGrid API returned error")
	}

	// Process each transaction
	for _, tx := range tronResp.Data {
		// Only process incoming transactions to our address
		if tx.To != address {
			continue
		}

		// Check if we've already processed this transaction
		var exists bool
		err := db.PostgresEngine.QueryRow(`
			SELECT EXISTS(SELECT 1 FROM crypto_payment_transactions WHERE transaction_hash = $1)
		`, tx.TransactionID).Scan(&exists)

		if err != nil {
			log.Printf("Error checking transaction existence: %v", err)
			continue
		}

		if exists {
			// Already processed this transaction
			continue
		}

		// Convert amount from string to float
		amount, err := strconv.ParseFloat(tx.Amount, 64)
		if err != nil {
			log.Printf("Error parsing amount: %v", err)
			continue
		}

		// Adjust for token decimals
		amount = amount / float64(10^tx.TokenInfo.Decimals)

		// Record the new transaction
		_, err = db.PostgresEngine.Exec(`
			INSERT INTO crypto_payment_transactions 
			(user_id, deposit_address_id, network_id, amount, token_symbol, transaction_hash, status, confirmations)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, userID, addressID, networkID, amount, tokenSymbol, tx.TransactionID, models.TransactionStatusConfirming, 1)

		if err != nil {
			log.Printf("Error recording transaction: %v", err)
			continue
		}

		log.Printf("Recorded new deposit: %s USDT from %s to %s (tx: %s)",
			tx.Amount, tx.From, address, tx.TransactionID)
	}

	return nil
}

// updatePendingTransactions checks confirmation status of pending transactions
func updatePendingTransactions() error {
	// Get all transactions in 'confirming' status
	rows, err := db.PostgresEngine.Query(`
		SELECT id, user_id, transaction_hash, amount, confirmations
		FROM crypto_payment_transactions
		WHERE status = $1
	`, models.TransactionStatusConfirming)

	if err != nil {
		return fmt.Errorf("failed to query pending transactions: %w", err)
	}
	defer rows.Close()

	// Process each transaction
	for rows.Next() {
		var txID, userID, confirmations int
		var txHash string
		var amount float64

		if err := rows.Scan(&txID, &userID, &txHash, &amount, &confirmations); err != nil {
			log.Printf("Error scanning transaction row: %v", err)
			continue
		}

		// Check current confirmation count from TronGrid
		currentConfirmations, err := getTransactionConfirmations(txHash)
		if err != nil {
			log.Printf("Error getting confirmations for tx %s: %v", txHash, err)
			continue
		}

		// Update confirmation count
		if currentConfirmations > confirmations {
			_, err = db.PostgresEngine.Exec(`
				UPDATE crypto_payment_transactions
				SET confirmations = $1
				WHERE id = $2
			`, currentConfirmations, txID)

			if err != nil {
				log.Printf("Error updating confirmations: %v", err)
				continue
			}

			log.Printf("Updated tx %s: %d confirmations", txHash, currentConfirmations)
		}

		// If we have enough confirmations, mark as confirmed and update user balance
		if currentConfirmations >= requiredConfirmations {
			if err := confirmTransaction(txID, userID, amount); err != nil {
				log.Printf("Error confirming transaction %d: %v", txID, err)
			}
		}
	}

	return nil
}

// getTransactionConfirmations checks the current confirmation count for a transaction
func getTransactionConfirmations(txHash string) (int, error) {
	// Get TronGrid API key from environment
	apiKey := os.Getenv("TRON_GRID_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("TRON_GRID_API_KEY not set")
	}

	// Query TronGrid for transaction info
	url := fmt.Sprintf("https://api.trongrid.io/v1/transactions/%s", txHash)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("TRON-PRO-API-KEY", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response to get confirmation count
	var result struct {
		BlockNumber int `json:"blockNumber"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	// Calculate confirmations based on current block height
	currentBlock, err := getCurrentBlockHeight()
	if err != nil {
		return 0, err
	}

	confirmations := currentBlock - result.BlockNumber
	if confirmations < 0 {
		confirmations = 0
	}

	return confirmations, nil
}

// getCurrentBlockHeight gets the current block height from TronGrid
func getCurrentBlockHeight() (int, error) {
	// Get TronGrid API key from environment
	apiKey := os.Getenv("TRON_GRID_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("TRON_GRID_API_KEY not set")
	}

	// Query TronGrid for current block
	url := "https://api.trongrid.io/wallet/getnowblock"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("TRON-PRO-API-KEY", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response to get block number
	var result struct {
		BlockHeader struct {
			RawData struct {
				Number int `json:"number"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.BlockHeader.RawData.Number, nil
}

// confirmTransaction marks a transaction as confirmed and updates the user's balance
func confirmTransaction(txID, userID int, amount float64) error {
	// Start a transaction
	tx, err := db.PostgresEngine.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Update transaction status
	_, err = tx.Exec(`
		UPDATE crypto_payment_transactions
		SET status = $1, confirmed_at = NOW()
		WHERE id = $2
	`, models.TransactionStatusConfirmed, txID)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	// Update user balance
	_, err = tx.Exec(`
		UPDATE webapp_users
		SET balance = balance + $1
		WHERE id = $2
	`, amount, userID)

	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Transaction %d confirmed: Added %.2f USDT to user %d balance", txID, amount, userID)
	return nil
}
