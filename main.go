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

type User struct {
	TelegramID   int     `json:"telegram_id"`
	TotalSpent   float64 `json:"total_spent"`
	Balance      float64 `json:"balance"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	Username     string  `json:"username"`
	LanguageCode string  `json:"language_code"`
}

type Node struct {
	AnyDeskAddress             string    `json:"any_desk_address"`
	Status                     string    `json:"status"`
	Software                   string    `json:"software"`
	Price                      float64   `json:"price"`
	Renter                     int       `json:"renter"`
	RentStartTime              time.Time `json:"rent_start_time"`
	LastBalanceUpdateTimestamp time.Time `json:"last_balance_update_timestamp"`
}

// Get all users
func getUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT telegram_id, total_spent, balance, first_name, last_name, username, language_code FROM users")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.TelegramID, &user.TotalSpent, &user.Balance, &user.FirstName, &user.LastName, &user.Username, &user.LanguageCode); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// Get nodes by renter
func getNodesByRenter(w http.ResponseWriter, r *http.Request) {
	renterID := r.URL.Query().Get("renter_id")
	if renterID == "" {
		http.Error(w, "renter_id is required", http.StatusBadRequest)
		return
	}

	rows, err := db.Query("SELECT any_desk_address, status, software, price, renter, rent_start_time, last_balance_update_timestamp FROM nodes WHERE renter = $1", renterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.AnyDeskAddress, &node.Status, &node.Software, &node.Price, &node.Renter, &node.RentStartTime, &node.LastBalanceUpdateTimestamp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		nodes = append(nodes, node)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
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

// Update node data
func updateNode(w http.ResponseWriter, r *http.Request) {
	var node Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `
		UPDATE nodes SET status=$1, software=$2, price=$3, renter=$4, rent_start_time=$5, last_balance_update_timestamp=$6
		WHERE any_desk_address=$7`
	_, err := db.Exec(query, node.Status, node.Software, node.Price, node.Renter, node.RentStartTime, node.LastBalanceUpdateTimestamp, node.AnyDeskAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// Create payment ticket
func createPaymentTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TelegramID int     `json:"telegram_id"`
		Amount     float64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO payments (user_id, amount, status, datetime) VALUES ($1, $2, 'unpaid', $3) RETURNING id`
	var paymentID int
	err := db.QueryRow(query, req.TelegramID, req.Amount, time.Now()).Scan(&paymentID)
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
	http.HandleFunc("/nodes", getNodesByRenter)
	http.HandleFunc("/user/update", updateUser)
	http.HandleFunc("/node/update", updateNode)
	http.HandleFunc("/payment/create", createPaymentTicket)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
