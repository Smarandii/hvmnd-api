package main

import (
	"hvmnd/api/db"
	"hvmnd/api/handlers"
	"log"
	"net/http"
)

func main() {
	db.InitDB()
	http.HandleFunc("GET /api/v1/ping", handlers.Ping)
	http.HandleFunc("GET /api/v1/users", handlers.GetUsers)
	http.HandleFunc("GET /api/v1/users/{id}", handlers.GetUsers)
	http.HandleFunc("POST /api/v1/users", handlers.CreateOrUpdateUser)

	http.HandleFunc("GET /api/v1/nodes", handlers.GetNodes)
	http.HandleFunc("GET /api/v1/nodes/{id}", handlers.GetNodes)
	http.HandleFunc("PATCH /api/v1/nodes", handlers.UpdateNode)

	http.HandleFunc("GET /api/v1/payments", handlers.GetPayments)
	http.HandleFunc("GET /api/v1/payments/{id}", handlers.GetPayments)
	http.HandleFunc("POST /api/v1/payments", handlers.CreatePaymentTicket)
	http.HandleFunc("PATCH /api/v1/payments/complete/{id}", handlers.CompletePayment)
	http.HandleFunc("PATCH /api/v1/payments/cancel/{id}", handlers.CancelPayment)

	log.Fatal(http.ListenAndServe(":9876", nil))
}
