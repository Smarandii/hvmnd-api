package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func generateSecureKey() (string, error) {
	key := make([]byte, 32) // 32 bytes for AES-256
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

func main() {
	key, err := generateSecureKey()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Generated key: %s\n", key)
	// Save this key and set it as CRYPTO_PRIVATE_KEYS_ENCRYPTION_KEY environment variable
}
