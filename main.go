package main

import (
	"context"
	"hvmnd/api/db"
	"hvmnd/api/handlers"
	middleware "hvmnd/api/middlewares"
	"hvmnd/api/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create a root context that can be canceled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up graceful shutdown
	go func() {
		// Listen for termination signals
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		log.Println("Shutting down gracefully...")
		cancel() // Cancel context to stop crypto monitor

		// Give ongoing operations a chance to complete (up to 5 seconds)
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()

	err := db.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Verify database connection is working
	if err := db.PostgresEngine.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connection established successfully")

	// Start the crypto monitor service
	services.StartCryptoMonitor(ctx)
	log.Println("Crypto monitor service started")

	// For public endpoint (like ping):
	http.Handle("GET /api/v1/ping", http.HandlerFunc(handlers.Ping))

	// Telegram handles:
	http.Handle("GET /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.GetTelegramUsers)))
	http.Handle("GET /api/v1/telegram/users/{id}", middleware.Auth(http.HandlerFunc(handlers.GetTelegramUsers)))
	http.Handle("POST /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.CreateTelegramUser)))
	http.Handle("PUT /api/v1/telegram/users", middleware.Auth(http.HandlerFunc(handlers.UpdateTelegramUser)))

	http.Handle("GET /api/v1/telegram/nodes", middleware.Auth(http.HandlerFunc(handlers.GetNodes)))
	http.Handle("GET /api/v1/telegram/nodes/{id}", middleware.Auth(http.HandlerFunc(handlers.GetNodes)))
	http.Handle("POST /api/v1/telegram/nodes", middleware.Auth(http.HandlerFunc(handlers.CreateNode)))
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
	http.Handle("POST /api/v1/telegram/user-interactions", middleware.Auth(http.HandlerFunc(handlers.CreateTgUserInteraction)))

	// WebApp handles:
	http.Handle("GET /api/v1/webapp/users", middleware.Auth(http.HandlerFunc(handlers.GetWebAppUsers)))
	http.Handle("GET /api/v1/webapp/users/{id}", middleware.Auth(http.HandlerFunc(handlers.GetWebAppUsers)))
	http.Handle("PATCH /api/v1/webapp/users", middleware.Auth(http.HandlerFunc(handlers.UpdateWebAppUser)))
	http.Handle("POST /api/v1/webapp/users", middleware.Auth(http.HandlerFunc(handlers.RegisterWebAppUser)))
	http.Handle("POST /api/v1/webapp/users/login", middleware.Auth(http.HandlerFunc(handlers.LoginWebAppUser)))
	http.Handle("GET /api/v1/webapp/users/confirm-email", middleware.Auth(http.HandlerFunc(handlers.ConfirmEmail)))

	http.Handle(
		"GET /api/v1/webapp/users/request-reset-password",
		middleware.Auth(
			http.HandlerFunc(handlers.RequestResetPassword),
		),
	)

	http.Handle(
		"POST /api/v1/webapp/users/reset-password",
		middleware.Auth(
			http.HandlerFunc(handlers.ResetPassword),
		),
	)

	// Common Entities handles:
	http.Handle("GET /api/v1/common/notifications", middleware.Auth(http.HandlerFunc(handlers.GetNotifications)))
	http.Handle("POST /api/v1/common/notifications", middleware.Auth(http.HandlerFunc(handlers.CreateNotification)))
	http.Handle("PUT /api/v1/common/notifications", middleware.Auth(http.HandlerFunc(handlers.UpdateNotification)))

	http.Handle("GET /api/v1/common/rent-sessions", middleware.Auth(http.HandlerFunc(handlers.GetRentSessions)))
	http.Handle("POST /api/v1/common/rent-sessions", middleware.Auth(http.HandlerFunc(handlers.CreateRentSession)))
	http.Handle("PATCH /api/v1/common/rent-sessions", middleware.Auth(http.HandlerFunc(handlers.UpdateRentSession)))

	// Crypto endpoints
	http.Handle("GET /api/v1/crypto/addresses", middleware.Auth(http.HandlerFunc(handlers.GetUserDepositAddresses)))
	http.Handle("POST /api/v1/crypto/addresses", middleware.Auth(http.HandlerFunc(handlers.CreateDepositAddress)))

	http.Handle("GET /api/v1/crypto/transactions", middleware.Auth(http.HandlerFunc(handlers.GetUserCryptoTransactions)))

	// Add these new routes for topup intents
	http.Handle("GET /api/v1/crypto/topup-intents", middleware.Auth(http.HandlerFunc(handlers.GetUserTopupIntents)))
	http.Handle("POST /api/v1/crypto/topup-intents", middleware.Auth(http.HandlerFunc(handlers.CreateTopupIntent)))
	http.Handle("PATCH /api/v1/crypto/topup-intents/cancel/{id}", middleware.Auth(http.HandlerFunc(handlers.CancelTopupIntent)))

	log.Println("Starting HTTP server on port 9876...")
	log.Fatal(http.ListenAndServe(":9876", nil))
}
