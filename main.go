package main

import (
	"hvmnd/api/db"
	"hvmnd/api/handlers"
	"log"
	"net/http"
)

func main() {
	db.InitDB()

	http.HandleFunc("GET /user", handlers.GetUsers)
	http.HandleFunc("GET /user/{id}", handlers.GetUsers)
	http.HandleFunc("POST /user", handlers.CreateOrUpdateUser)

	http.HandleFunc("GET /node", handlers.GetNodes)
	http.HandleFunc("GET /node/{id}", handlers.GetNodes)
	http.HandleFunc("PATCH /node", handlers.UpdateNode)

	http.HandleFunc("GET /payment", handlers.GetPayments)
	http.HandleFunc("GET /payment/{id}", handlers.GetPayments)
	http.HandleFunc("POST /payment", handlers.CreatePaymentTicket)
	http.HandleFunc("PATCH /payment/complete/{id}", handlers.CompletePayment)
	http.HandleFunc("PATCH /payment/cancel/{id}", handlers.CancelPayment)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
