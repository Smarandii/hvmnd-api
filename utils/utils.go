package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"hvmnd/api/models"
	"log"
	"net/http"
)

func WriteJSONResponse(w http.ResponseWriter, statusCode int, response models.APIResponse) {
	log.Printf("%s", response.Message)

	if response.Success {
		log.Printf("%s", response.Error)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func HvmndGenerateHash(question, answer string) string {
	data := question + answer
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])[:32] // Take the first 32 characters of the hex string
}

func GenerateRandomToken(length int) string {
	// If length is an odd number, add 1 to make it even so hex encoding is consistent
	if length%2 != 0 {
		length++
	}

	// length/2 bytes generate length hex chars
	b := make([]byte, length/2)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(b)
}
