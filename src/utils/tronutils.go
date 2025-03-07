package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"crypto/ecdsa"
	"encoding/hex"

	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/mr-tron/base58"

	"github.com/ethereum/go-ethereum/crypto"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

// CreateAccountResponse represents the response from TronGrid's CreateAccount API.
type CreateAccountResponse struct {
	RawData struct {
		Contract []struct {
			Parameter struct {
				Value struct {
					OwnerAddress   string `json:"owner_address"`
					AccountAddress string `json:"account_address"`
				} `json:"value"`
			} `json:"parameter"`
		} `json:"contract"`
	} `json:"raw_data"`
	TxID string `json:"txID"`
}

func s256(s []byte) []byte {
	h := sha256.New()
	h.Write(s)
	bs := h.Sum(nil)
	return bs
}

func generateTronAddressLocally() (string, string, error) {
	// Generate a private key
	privateKeyECDSA, err := crypto.GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	privateKeyBytes := crypto.FromECDSA(privateKeyECDSA)
	privateKeyHex := hexutil.Encode(privateKeyBytes)[2:]

	publicKey := privateKeyECDSA.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
	address = "41" + address[2:]
	fmt.Println("address hex: ", address)
	addb, _ := hex.DecodeString(address)
	hash1 := s256(s256(addb))
	secret := hash1[:4]
	addb = append(addb, secret...)

	return base58.Encode(addb), privateKeyHex, nil
}

func activateTronAddressViaTronGridAPI(newAccountAddress string) (bool, error) {
	// Load required environment variables
	ownerAddress := os.Getenv("TRON_OWNER_ADDRESS")
	if ownerAddress == "" {
		return false, fmt.Errorf("TRON_OWNER_ADDRESS not set")
	}
	tronGridAPIKey := os.Getenv("TRON_GRID_API_KEY")
	if tronGridAPIKey == "" {
		return false, fmt.Errorf("TRON_GRID_API_KEY not set")
	}

	// Prepare the payload as per TronGrid documentation.
	payloadMap := map[string]interface{}{
		"owner_address":   ownerAddress,
		"account_address": newAccountAddress,
		"visible":         true,
	}
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return false, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Use the appropriate TronGrid endpoint.
	// For testing, we're using the Shasta test network endpoint.
	tronGridURL := "https://api.shasta.trongrid.io/wallet/createaccount"

	req, err := http.NewRequestWithContext(context.Background(), "POST", tronGridURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TRON-PRO-API-KEY", tronGridAPIKey)

	// Create a custom HTTP client with TLS verification disabled
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		return false, fmt.Errorf("non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var createResp CreateAccountResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}
	log.Printf("Activation transaction ID: %s", createResp.TxID)
	return true, nil
}

// New function to encrypt private key
func EncryptPrivateKey(privateKey string, encodedEncryptionKey string) (string, error) {
	// Decode the Base64 encryption key first
	encryptionKey, err := base64.StdEncoding.DecodeString(encodedEncryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode encryption key: %w", err)
	}

	// Ensure the key is exactly 32 bytes (for AES-256)
	if len(encryptionKey) != 32 {
		return "", fmt.Errorf("encryption key must be exactly 32 bytes (got %d)", len(encryptionKey))
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(privateKey), nil)
	return hex.EncodeToString(ciphertext), nil
}

func GenerateTRC20DepositWalletAddress() (string, string, error) {
	// GenerateTRC20DepositWalletAddress generates a new Tron account locally
	// and then activates it via TronGrid.
	// Returns address and encrypted private key

	encryptionKey := os.Getenv("CRYPTO_PRIVATE_KEYS_ENCRYPTION_KEY")
	if encryptionKey == "" {
		return "", "", fmt.Errorf("CRYPTO_PRIVATE_KEYS_ENCRYPTION_KEY not set")
	}

	newAccountAddress, privateKey, err := generateTronAddressLocally()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate Tron address: %w", err)
	}
	log.Printf("Locally generated new account address: %s", newAccountAddress)

	// Step 2: Activate the new account via TronGrid API.
	success, err := activateTronAddressViaTronGridAPI(newAccountAddress)
	if err != nil || !success {
		return "", "", fmt.Errorf("failed to activate Tron address: %w", err)
	}

	encryptedPrivateKey, err := EncryptPrivateKey(privateKey, encryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to encrypt private key: %w", err)
	}

	return newAccountAddress, encryptedPrivateKey, nil
}
