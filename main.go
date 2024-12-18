package main

import (
	"hvmnd/api/db"
	"hvmnd/api/handlers"
	middleware "hvmnd/api/middlewares"
	"log"
	"net/http"
)

func main() {
	db.InitDB()

	// For public endpoint (like ping):
	http.Handle("GET /api/v1/ping", http.HandlerFunc(handlers.Ping))

	// Telegram handles:
	http.Handle("GET /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.GetTelegramUsers)))
	http.Handle("GET /api/v1/telegram/users/{id}", middleware.Auth(http.HandlerFunc(handlers.GetTelegramUsers)))
	http.Handle("POST /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.CreateTelegramUser)))
	http.Handle("PUT /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.UpdateTelegramUser)))

	http.Handle("GET /api/v1/telegram/nodes", middleware.Auth(http.HandlerFunc(handlers.GetNodes)))
	http.Handle("GET /api/v1/telegram/nodes/{id}", middleware.Auth(http.HandlerFunc(handlers.GetNodes)))
	http.Handle("PATCH /api/v1/telegram/nodes", middleware.Auth(http.HandlerFunc(handlers.UpdateNode)))

	http.Handle("GET /api/v1/telegram/payments", middleware.Auth(http.HandlerFunc(handlers.GetPayments)))
	http.Handle("GET /api/v1/telegram/payments/{id}", middleware.Auth(http.HandlerFunc(handlers.GetPayments)))
	http.Handle("POST /api/v1/telegram/payments", middleware.Auth(http.HandlerFunc(handlers.CreatePaymentTicket)))
	http.Handle("PATCH /api/v1/telegram/payments/complete/{id}", middleware.Auth(http.HandlerFunc(handlers.CompletePayment)))
	http.Handle("PATCH /api/v1/telegram/payments/cancel/{id}", middleware.Auth(http.HandlerFunc(handlers.CancelPayment)))

	http.Handle("POST /api/v1/telegram/quiz/save-hash", middleware.Auth(http.HandlerFunc(handlers.SaveHashMapping)))
	http.Handle("GET /api/v1/telegram/quiz/get-question-answer", middleware.Auth(http.HandlerFunc(handlers.GetQuestionAnswerByHash)))
	http.Handle("POST /api/v1/telegram/quiz/save-answer", middleware.Auth(http.HandlerFunc(handlers.SaveUserAnswer)))

	http.Handle("POST /api/v1/telegram/tokens", middleware.Auth(http.HandlerFunc(handlers.CreateToken)))
	http.Handle("GET /api/v1/telegram/tokens", middleware.Auth(http.HandlerFunc(handlers.GetTokens)))

	// WebApp handles:
	http.Handle("GET /api/v1/webapp/users", middleware.Auth(http.HandlerFunc(handlers.GetWebAppUsers)))
	http.Handle("GET /api/v1/webapp/users/{id}", middleware.Auth(http.HandlerFunc(handlers.GetWebAppUsers)))
	http.Handle("POST /api/v1/webapp/users", middleware.Auth(http.HandlerFunc(handlers.RegisterWebAppUser)))
	http.Handle("POST /api/v1/webapp/users/login", middleware.Auth(http.HandlerFunc(handlers.LoginWebAppUser)))
	http.Handle("GET /api/v1/webapp/users/confirm-email", middleware.Auth(http.HandlerFunc(handlers.ConfirmEmail)))

	log.Fatal(http.ListenAndServe(":9876", nil))
}
