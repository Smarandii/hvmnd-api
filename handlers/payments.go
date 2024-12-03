package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hvmnd/api/db"
	"hvmnd/api/models"
	"net/http"
	"time"
)

func GetPayments(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}
	userID := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	limit := r.URL.Query().Get("limit")

	query := `
		SELECT id, user_id, amount, status, datetime FROM payments WHERE 1=1
	`
	var args []interface{}
	argIndex := 1

	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}
	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if limit != "" {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	rows, err := db.PostgresEngine.Query(query, args...)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to fetch payments: " + err.Error(),
		})
		return
	}
	defer rows.Close()

	var payments []models.Payment
	for rows.Next() {
		var payment models.Payment
		err := rows.Scan(
			&payment.ID,
			&payment.UserID,
			&payment.Amount,
			&payment.Status,
			&payment.Datetime,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payments = append(payments, payment)
	}

	if len(payments) == 0 {
		writeJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "No payments found matching the criteria",
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Found %d payments", len(payments)),
		Data:    payments,
	})
}

func CreatePaymentTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int     `json:"user_id"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Amount <= 0 {
		writeJSONResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   "Amount must be greater than 0",
		})
		return
	}

	// Check if user exists
	var exists bool
	err := db.PostgresEngine.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", req.UserID).Scan(&exists)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to check user existence: " + err.Error(),
		})
		return
	}

	if !exists {
		writeJSONResponse(w, http.StatusNotFound, APIResponse{
			Success: false,
			Error:   "User not found",
		})
		return
	}

	query := `
		INSERT INTO payments (user_id, amount, status, datetime) 
		VALUES ($1, $2, 'unpaid', $3) RETURNING id
	`
	var paymentID int
	err = db.PostgresEngine.QueryRow(query, req.UserID, req.Amount, time.Now()).Scan(&paymentID)
	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to create payment ticket: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusCreated, APIResponse{
		Success: true,
		Message: "Payment ticket created successfully",
		Data: map[string]int{
			"payment_ticket_id": paymentID,
		},
	})
}

func CompletePayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}

	var amount float64
	var userID int
	var currentStatus string

	// First, check the current payment status
	query := `
		SELECT amount, user_id, status 
		FROM payments 
		WHERE id=$1
	`
	err := db.PostgresEngine.QueryRow(query, id).Scan(&amount, &userID, &currentStatus)

	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONResponse(w, http.StatusNotFound, APIResponse{
				Success: false,
				Error:   "Payment not found",
			})
			return
		}
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	// If the payment is already marked as "paid", return early and do nothing
	if currentStatus == "paid" {
		writeJSONResponse(w, http.StatusAlreadyReported, APIResponse{
			Success: true,
			Message: "Payment already completed",
			Data: map[string]string{
				"payment_ticket_id": id,
				"status":            "paid",
			},
		})
		return
	}

	// Mark the payment as "paid"
	query = `
		UPDATE payments SET
		status=$1
		WHERE id=$2
	`
	_, err = db.PostgresEngine.Exec(query, "paid", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Update the user's balance
	query = `
		UPDATE users SET
		balance = balance + $1
		WHERE id=$2
	`
	_, err = db.PostgresEngine.Exec(query, amount, userID)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Payment completed successfully",
		Data: map[string]string{
			"payment_ticket_id": id,
		},
	})
}

func CancelPayment(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.PathValue("id")
	}

	var amount float64
	var userID int
	var currentStatus string

	// First, check the current payment status
	query := `
		SELECT amount, user_id, status 
		FROM payments 
		WHERE id=$1
	`
	err := db.PostgresEngine.QueryRow(query, id).Scan(&amount, &userID, &currentStatus)

	if err != nil {
		if err == sql.ErrNoRows {
			writeJSONResponse(w, http.StatusNotFound, APIResponse{
				Success: false,
				Error:   "Payment not found",
			})
			return
		}
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to fetch payment: " + err.Error(),
		})
		return
	}

	// If the payment is already "cancelled", return early
	if currentStatus == "cancelled" {
		writeJSONResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Payment already cancelled",
			Data: map[string]string{
				"payment_ticket_id": id,
				"status":            "cancelled",
			},
		})
		return
	}

	// If the payment was "paid", adjust the user's balance
	if currentStatus == "paid" {
		query = `
			UPDATE users SET
			balance = balance - $1
			WHERE id=$2
		`
		_, err = db.PostgresEngine.Exec(query, amount, userID)

		if err != nil {
			writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
				Success: false,
				Error:   "Failed to update user balance: " + err.Error(),
			})
			return
		}
	}

	// Mark the payment as "cancelled"
	query = `
		UPDATE payments SET
		status=$1
		WHERE id=$2
	`
	_, err = db.PostgresEngine.Exec(query, "cancelled", id)

	if err != nil {
		writeJSONResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   "Failed to cancel payment: " + err.Error(),
		})
		return
	}

	writeJSONResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Payment cancelled successfully",
		Data: map[string]string{
			"payment_ticket_id": id,
			"status":            "cancelled",
		},
	})
}
