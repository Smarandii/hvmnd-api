package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Database initialization
func initDB() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get the connection string from the environment variable
	connStr := os.Getenv("POSTGRES_URL")
	if connStr == "" {
		log.Fatal("POSTGRES_URL is not set in the .env file")
	}

	// Open a database connection
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the database to ensure connection
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database connection established")
}

// Helper function to return either null or string value
func nullStringOrValue(ns sql.NullString) interface{} {
	if ns.Valid {
		return ns.String
	}
	return nil
}

// Helper function to return either null or time value
func nullTimeOrValue(nt sql.NullTime) interface{} {
	if nt.Valid {
		return nt.Time
	}
	return nil
}

// Helper function to return either null or Int32 value
func nullInt32OrValue(nt sql.NullInt32) interface{} {
	if nt.Valid {
		return nt.Int32
	}
	return nil
}

// Helper function to return either null or Int16 value
func nullInt16OrValue(nt sql.NullInt16) interface{} {
	if nt.Valid {
		return nt.Int16
	}
	return nil
}

// User structure with custom JSON marshalling
type User struct {
	ID           int            `json:"id"`
	TelegramID   int            `json:"telegram_id"`
	TotalSpent   float64        `json:"total_spent"`
	Balance      float64        `json:"balance"`
	FirstName    sql.NullString `json:"-"`
	LastName     sql.NullString `json:"-"`
	Username     sql.NullString `json:"-"`
	LanguageCode sql.NullString `json:"-"`
}

// Custom JSON marshalling for the User struct to handle sql.NullString
func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		FirstName    interface{} `json:"first_name"`
		LastName     interface{} `json:"last_name"`
		Username     interface{} `json:"username"`
		LanguageCode interface{} `json:"language_code"`
		Alias
	}{
		FirstName:    nullStringOrValue(u.FirstName),
		LastName:     nullStringOrValue(u.LastName),
		Username:     nullStringOrValue(u.Username),
		LanguageCode: nullStringOrValue(u.LanguageCode),
		Alias:        (Alias)(u),
	})
}

// Node structure with all fields matching the DDL, and custom JSON marshalling
type Node struct {
	ID                         int            `json:"id"`
	OldID                      sql.NullInt32  `json:"old_id"`
	AnyDeskAddress             string         `json:"any_desk_address"`
	AnyDeskPassword            string         `json:"any_desk_password"`
	Status                     string         `json:"status"`
	Software                   sql.NullString `json:"software"`
	Price                      float64        `json:"price"`
	Renter                     sql.NullInt16  `json:"renter"`
	RentStartTime              sql.NullTime   `json:"rent_start_time"`
	LastBalanceUpdateTimestamp sql.NullTime   `json:"last_balance_update_timestamp"`
	CPU                        sql.NullString `json:"cpu"`
	GPU                        sql.NullString `json:"gpu"`
	OtherSpecs                 sql.NullString `json:"other_specs"`
	Licenses                   sql.NullString `json:"licenses"`
	MachineID                  sql.NullString `json:"machine_id"`
}

// Custom JSON marshalling for Node struct to handle nullable fields
func (n Node) MarshalJSON() ([]byte, error) {
	type Alias Node
	return json.Marshal(&struct {
		OldID                      interface{} `json:"old_id"`
		Software                   interface{} `json:"software"`
		Renter                     interface{} `json:"renter"`
		RentStartTime              interface{} `json:"rent_start_time"`
		LastBalanceUpdateTimestamp interface{} `json:"last_balance_update_timestamp"`
		CPU                        interface{} `json:"cpu"`
		GPU                        interface{} `json:"gpu"`
		OtherSpecs                 interface{} `json:"other_specs"`
		Licenses                   interface{} `json:"licenses"`
		MachineID                  interface{} `json:"machine_id"`
		Alias
	}{
		OldID:                      nullInt32OrValue(n.OldID),
		Software:                   nullStringOrValue(n.Software),
		Renter:                     nullInt16OrValue(n.Renter),
		RentStartTime:              nullTimeOrValue(n.RentStartTime),
		LastBalanceUpdateTimestamp: nullTimeOrValue(n.LastBalanceUpdateTimestamp),
		CPU:                        nullStringOrValue(n.CPU),
		GPU:                        nullStringOrValue(n.GPU),
		OtherSpecs:                 nullStringOrValue(n.OtherSpecs),
		Licenses:                   nullStringOrValue(n.Licenses),
		MachineID:                  nullStringOrValue(n.MachineID),
		Alias:                      (Alias)(n),
	})
}

type Payment struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	Amount   float64   `json:"amount"`
	Status   string    `json:"status"`
	Datetime time.Time `json:"datetime"`
}

// Get users by optional filters (ID or Telegram ID)
func getUsers(w http.ResponseWriter, r *http.Request) {
	telegramID := r.URL.Query().Get("telegram_id")
	id := r.URL.Query().Get("id")
	limit := r.URL.Query().Get("limit")

	query := "SELECT id, telegram_id, total_spent, balance, first_name, last_name, username, language_code FROM users WHERE 1=1"
	var args []interface{}
	argIndex := 1

	if telegramID != "" {
		query += fmt.Sprintf(" AND telegram_id = $%d", argIndex)
		args = append(args, telegramID)
		argIndex++
	}
	if id != "" {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, id)
		argIndex++
	}
	if limit != "" {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.TelegramID, &user.TotalSpent, &user.Balance, &user.FirstName, &user.LastName, &user.Username, &user.LanguageCode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Save user data (Create or Update)
func createUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		INSERT INTO users (telegram_id, total_spent, balance, first_name, last_name, username, language_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (telegram_id) DO UPDATE
		SET total_spent = EXCLUDED.total_spent, balance = EXCLUDED.balance, first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, username = EXCLUDED.username, language_code = EXCLUDED.language_code;
	`
	_, err := db.Exec(query, user.TelegramID, user.TotalSpent, user.Balance, user.FirstName, user.LastName, user.Username, user.LanguageCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Update user data
func updateUser(w http.ResponseWriter, r *http.Request) {
	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE users SET total_spent=$1, balance=$2, first_name=$3, last_name=$4, username=$5, language_code=$6 
		WHERE telegram_id=$7`
	_, err := db.Exec(query, user.TotalSpent, user.Balance, user.FirstName, user.LastName, user.Username, user.LanguageCode, user.TelegramID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Get nodes by optional filters
func getNodes(w http.ResponseWriter, r *http.Request) {
	renter := r.URL.Query().Get("renter")
	status := r.URL.Query().Get("status")
	anyDeskAddress := r.URL.Query().Get("anydesk_address")
	software := r.URL.Query().Get("software")

	query := "SELECT id, old_id, any_desk_address, any_desk_password, status, software, price, renter, rent_start_time, last_balance_update_timestamp, cpu, gpu, other_specs, licenses, machine_id FROM nodes WHERE 1=1"
	var args []interface{}
	argIndex := 1

	if renter != "" {
		query += fmt.Sprintf(" AND renter = $%d", argIndex)
		args = append(args, renter)
		argIndex++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if anyDeskAddress != "" {
		query += fmt.Sprintf(" AND any_desk_address = $%d", argIndex)
		args = append(args, anyDeskAddress)
		argIndex++
	}
	if software != "" {
		query += fmt.Sprintf(" AND software ILIKE $%d", argIndex)
		args = append(args, "%"+software+"%")
		argIndex++
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.ID, &node.OldID, &node.AnyDeskAddress, &node.AnyDeskPassword, &node.Status, &node.Software, &node.Price, &node.Renter, &node.RentStartTime, &node.LastBalanceUpdateTimestamp, &node.CPU, &node.GPU, &node.OtherSpecs, &node.Licenses, &node.MachineID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nodes = append(nodes, node)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}

// Update node data
func updateNode(w http.ResponseWriter, r *http.Request) {
	var node Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE nodes SET status=$1, software=$2, price=$3, renter=$4, rent_start_time=$5, last_balance_update_timestamp=$6, cpu=$7, gpu=$8, other_specs=$9, licenses=$10, machine_id=$11
		WHERE any_desk_address=$12`
	_, err := db.Exec(query, node.Status, node.Software, node.Price, node.Renter, node.RentStartTime, node.LastBalanceUpdateTimestamp, node.CPU, node.GPU, node.OtherSpecs, node.Licenses, node.MachineID, node.AnyDeskAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Get payments by optional filters (ID, User ID, or Status)
func getPayments(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	status := r.URL.Query().Get("status")
	id := r.URL.Query().Get("id")
	limit := r.URL.Query().Get("limit")

	query := "SELECT id, user_id, amount, status, datetime FROM payments WHERE 1=1"
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
		// Using exact match for status with lowercase comparison for safety
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if limit != "" {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	fmt.Printf("Query: %s\n", query)
	fmt.Printf("Args: %v\n", args)

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		if err := rows.Scan(&payment.ID, &payment.UserID, &payment.Amount, &payment.Status, &payment.Datetime); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		payments = append(payments, payment)
	}

	w.Header().Set("Content-Type", "application/json")
	if len(payments) == 0 {
		json.NewEncoder(w).Encode(nil) // Return null if no payments are found
	} else {
		json.NewEncoder(w).Encode(payments)
	}
}

// Create payment ticket
func createPaymentTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID int     `json:"user_id"`
		Amount float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO payments (user_id, amount, status, datetime) VALUES ($1, $2, 'unpaid', $3) RETURNING id`
	var paymentID int
	err := db.QueryRow(query, req.UserID, req.Amount, time.Now()).Scan(&paymentID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"payment_ticket_id": paymentID})
}

func main() {
	initDB()

	http.HandleFunc("/users", getUsers)
	http.HandleFunc("/user/create", createUser)
	http.HandleFunc("/user/update", updateUser)

	http.HandleFunc("/nodes", getNodes)
	http.HandleFunc("/node/update", updateNode)

	http.HandleFunc("/payments", getPayments)
	http.HandleFunc("/payment/create", createPaymentTicket)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
