package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"hvmnd/api/db"
	"hvmnd/api/models"
)

const (
	// How often to check for new transactions
	pollingInterval = 30 * time.Second

	// Number of confirmations required to consider a transaction confirmed
	requiredConfirmations = 6
)

// TronGridTransferResponse represents the response from TronGrid API for TRC-20 transfers
type TronGridTransferResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		TransactionID string `json:"transaction_id"`
		Block         int    `json:"block_timestamp"`
		From          string `json:"from"`
		To            string `json:"to"`
		Amount        string `json:"value"`
		Type          string `json:"type"`
		TokenInfo     struct {
			Name     string `json:"name"`
			Symbol   string `json:"symbol"`
			Decimals int    `json:"decimals"`
			Address  string `json:"address"`
		} `json:"token_info"`
	} `json:"data"`
	Meta struct {
		PageSize int `json:"page_size"`
		At       int `json:"at"`
	} `json:"meta"`
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
				if err := checkPendingTopupIntents(); err != nil {
					log.Printf("Error checking pending topup intents: %v", err)
				}
				if err := updateConfirmingIntents(); err != nil {
					log.Printf("Error updating confirming intents: %v", err)
				}
			}
		}
	}()
}

// checkPendingTopupIntents checks for deposits matching pending topup intents
func checkPendingTopupIntents() error {
	// Get all active (unexpired) pending topup intents
	rows, err := db.PostgresEngine.Query(`
		SELECT i.id, i.user_id, i.network_id, i.amount, 
		       n.contract_address, n.token_symbol, n.name,
		       a.address
		FROM crypto_topup_intents i
		JOIN crypto_networks n ON i.network_id = n.id
		JOIN crypto_deposit_addresses a ON a.user_id = i.user_id AND a.network_id = i.network_id
		WHERE i.status = $1 
		  AND i.expires_at > NOW()
		  AND a.is_used = false
	`, models.TopupIntentStatusPending)

	if err != nil {
		return fmt.Errorf("failed to query pending topup intents: %w", err)
	}
	defer rows.Close()

	// Process each pending intent
	for rows.Next() {
		var intentID, userID, networkID int
		var amount float64
		var contractAddress, networkName, depositAddress string

		if err := rows.Scan(&intentID, &userID, &networkID, &amount,
			&contractAddress, &networkName,
			&depositAddress); err != nil {
			log.Printf("Error scanning intent row: %v", err)
			continue
		}

		// Only support TRC-20 for now
		if networkName != "TRC-20" {
			continue
		}

		// Check for transactions matching this intent
		if err := checkIntentTransactions(intentID, depositAddress,
			contractAddress, amount); err != nil {
			log.Printf("Error checking transactions for intent %d: %v", intentID, err)
		}

		// Wait before proceeding to the next intent to avoid overloading TronGrid with requests
		time.Sleep(5 * time.Second)
	}

	return nil
}

// checkIntentTransactions queries TronGrid for transactions matching a specific topup intent
func checkIntentTransactions(intentID int, depositAddress, contractAddress string,
	expectedAmount float64) error {
	// Get TronGrid API key from environment
	apiKey := os.Getenv("TRON_GRID_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("TRON_GRID_API_KEY not set")
	}

	// Construct TronGrid API URL for TRC-20 transfers
	url := fmt.Sprintf("https://api.trongrid.io/v1/accounts/%s/transactions/trc20?contract_address=%s&only_confirmed=true&limit=20",
		depositAddress, contractAddress)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add API key header
	req.Header.Add("TRON-PRO-API-KEY", apiKey)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

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
		if tx.To != depositAddress {
			log.Printf("Skip outcoming transaction...")
			continue
		}

		// Check if we've already processed this transaction
		var exists bool
		err := db.PostgresEngine.QueryRow(`
			SELECT EXISTS(
				SELECT 1 FROM crypto_topup_intents 
				WHERE transaction_hash = $1 AND status IN ($2, $3, $4)
			)
		`, tx.TransactionID,
			models.TopupIntentStatusConfirming,
			models.TopupIntentStatusConfirmed,
			models.TopupIntentStatusFailed).Scan(&exists)

		if err != nil {
			log.Printf("Error checking transaction existence: %v", err)
			continue
		}

		if exists {
			log.Printf("Skip already processed transaction...")
			continue
		}

		// Convert amount from string to float
		amount, err := strconv.ParseFloat(tx.Amount, 64)
		if err != nil {
			log.Printf("Error parsing amount: %v", err)
			continue
		}

		// Adjust for token decimals
		amount = amount / math.Pow(10, float64(tx.TokenInfo.Decimals))

		// Check if the amount matches what was expected
		// We could implement a tolerance here if needed
		if amount != expectedAmount {
			log.Printf("Transaction amount %.2f doesn't match expected amount %.2f",
				amount, expectedAmount)
			continue
		}

		// Update the intent with the transaction hash and mark as confirming
		_, err = db.PostgresEngine.Exec(`
			UPDATE crypto_topup_intents
			SET status = $1, transaction_hash = $2
			WHERE id = $3
		`, models.TopupIntentStatusConfirming, tx.TransactionID, intentID)

		if err != nil {
			log.Printf("Error updating topup intent: %v", err)
			continue
		}

		log.Printf("Found matching transaction for intent %d: %s USDT from %s to %s (tx: %s)",
			intentID, tx.Amount, tx.From, depositAddress, tx.TransactionID)
	}

	return nil
}

// updateConfirmingIntents checks confirmation status of intents in confirming status
func updateConfirmingIntents() error {
	// Get all intents in 'confirming' status
	rows, err := db.PostgresEngine.Query(`
		SELECT i.id, i.user_id, i.network_id, i.amount, i.transaction_hash
		FROM crypto_topup_intents i
		WHERE i.status = $1
	`, models.TopupIntentStatusConfirming)

	if err != nil {
		return fmt.Errorf("failed to query confirming intents: %w", err)
	}
	defer rows.Close()

	// Process each intent
	for rows.Next() {
		var intentID, userID, networkID int
		var amount float64
		var txHash string

		if err := rows.Scan(&intentID, &userID, &networkID, &amount, &txHash); err != nil {
			log.Printf("Error scanning intent row: %v", err)
			continue
		}

		// Check current confirmation count from TronGrid
		confirmations, err := getTransactionConfirmations(txHash)
		if err != nil {
			log.Printf("Error getting confirmations for tx %s: %v", txHash, err)
			continue
		}

		log.Printf("Intent %d transaction %s has %d confirmations",
			intentID, txHash, confirmations)

		// If we have enough confirmations, process the deposit
		if confirmations >= requiredConfirmations {
			if err := processConfirmedDeposit(intentID, userID, networkID, amount, txHash); err != nil {
				log.Printf("Error processing confirmed deposit for intent %d: %v", intentID, err)
			}
		}
	}

	return nil
}

// processConfirmedDeposit handles a confirmed deposit:
// 1. Creates a CryptoPaymentTransaction record
// 2. Updates the user's balance
// 3. Marks the topup intent as confirmed
func processConfirmedDeposit(intentID, userID, networkID int, amount float64, txHash string) error {
	// Start a database transaction
	tx, err := db.PostgresEngine.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the deposit address ID
	var depositAddressID int
	err = tx.QueryRow(`
		SELECT id FROM crypto_deposit_addresses 
		WHERE user_id = $1 AND network_id = $2 AND is_used = false
		LIMIT 1
	`, userID, networkID).Scan(&depositAddressID)

	if err != nil {
		return fmt.Errorf("failed to get deposit address ID: %w", err)
	}

	// 1. Create a crypto_payment_transaction record
	_, err = tx.Exec(`
		INSERT INTO crypto_payment_transactions
		(user_id, deposit_address_id, network_id, amount, transaction_hash, 
		 status, confirmations, created_at, confirmed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
	`, userID, depositAddressID, networkID, amount, txHash,
		models.TransactionStatusConfirmed, requiredConfirmations)

	if err != nil {
		return fmt.Errorf("failed to create payment transaction record: %w", err)
	}

	// 2. Update user balance
	_, err = tx.Exec(`
		UPDATE webapp_users
		SET balance = balance + $1
		WHERE id = $2
	`, amount, userID)

	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	// 3. Update the topup intent status
	_, err = tx.Exec(`
		UPDATE crypto_topup_intents
		SET status = $1, confirmed_at = NOW()
		WHERE id = $2
	`, models.TopupIntentStatusConfirmed, intentID)

	if err != nil {
		return fmt.Errorf("failed to update intent status: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("Successfully processed deposit for intent %d: Added %.2f USDT to user %d balance",
		intentID, amount, userID)
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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}

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
