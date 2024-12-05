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
	http.HandleFunc("GET /api/v1/user", handlers.GetUsers)
	http.HandleFunc("GET /api/v1/user/{id}", handlers.GetUsers)
	http.HandleFunc("POST /api/v1/user", handlers.CreateOrUpdateUser)

	http.HandleFunc("GET /api/v1/node", handlers.GetNodes)
	http.HandleFunc("GET /api/v1/node/{id}", handlers.GetNodes)
	http.HandleFunc("PATCH /api/v1/node", handlers.UpdateNode)

	http.HandleFunc("GET /api/v1/payment", handlers.GetPayments)
	http.HandleFunc("GET /api/v1/payment/{id}", handlers.GetPayments)
	http.HandleFunc("POST /api/v1/payment", handlers.CreatePaymentTicket)
	http.HandleFunc("PATCH /api/v1/payment/complete/{id}", handlers.CompletePayment)
	http.HandleFunc("PATCH /api/v1/payment/cancel/{id}", handlers.CancelPayment)

	log.Fatal(http.ListenAndServe(":9876", nil))
}
