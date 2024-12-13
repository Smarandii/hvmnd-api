package middleware

import (
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := godotenv.Load()
		if err != nil {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Auth error loading .env file.",
				Error:   err.Error(),
			})
			return
		}

		apiToken := os.Getenv("API_TOKEN") // Load token from environment
		log.Printf("Loaded API_TOKEN: %q", apiToken)

		if apiToken == "" {
			// If no token is set on the server, deny all requests or panic
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Server misconfiguration: No API_TOKEN set.",
			})
			return
		}

		// Check for Authorization header
		authHeader := r.Header.Get("Authorization")
		log.Printf("Received Authorization header: %q", authHeader)
		if authHeader == "" {
			utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Missing Authorization header",
			})
			return
		}

		log.Printf("apiToken from env: %q", apiToken)
		log.Printf("Authorization header: %q", authHeader)

		// Expecting "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != apiToken {
			utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Invalid or missing token",
			})
			return
		}

		// Token is valid, proceed
		next.ServeHTTP(w, r)
	})
}
