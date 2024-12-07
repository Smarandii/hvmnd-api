package handlers

import (
	"net/http"
	"hvmnd/api/db"
	"encoding/json"
	"hvmnd/api/utils"
)

func SaveHashMapping(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Question string `json:"question"`
		Answer   string `json:"answer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		return
	}

	// Generate the hash
	hash := utils.GenerateHash(input.Question, input.Answer)

	query := `
		INSERT INTO quiz_hash_map (hash, question, answer)
		VALUES ($1, $2, $3)
		ON CONFLICT (hash) DO NOTHING;
	`

	_, err := db.PostgresEngine.Exec(query, hash, input.Question, input.Answer)
	if err != nil {
		http.Error(w, "Failed to save hash mapping: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Hash mapping saved successfully",
		Data: map[string]string{"hash": hash},
	})
}

func GetQuestionAnswerByHash(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	query := `
		SELECT question, answer
		FROM quiz_hash_map
		WHERE hash = $1;
	`

	var question, answer string
	err := db.PostgresEngine.QueryRow(query, hash).Scan(&question, &answer)
	if err != nil {
		http.Error(w, "Hash not found: "+err.Error(), http.StatusNotFound)
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]string{
			"question": question,
			"answer":   answer,
		},
	})
}

func SaveUserAnswer(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TelegramID int    `json:"telegram_id"`
		Question   string `json:"question"`
		Answer     string `json:"answer"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid input format", http.StatusBadRequest)
		return
	}

	// Generate the hash
	hash := utils.GenerateHash(input.Question, input.Answer)

	query := `
		INSERT INTO quiz_answers (telegram_id, question, answer, hash)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (telegram_id, question) DO UPDATE
		SET answer = EXCLUDED.answer, hash = EXCLUDED.hash;
	`

	_, err := db.PostgresEngine.Exec(query, input.TelegramID, input.Question, input.Answer, hash)
	if err != nil {
		http.Error(w, "Failed to save user answer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "User answer saved successfully",
		Data: map[string]string{"hash": hash},
	})
}
