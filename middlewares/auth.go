package middleware

import (
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"log"
	"net/http"
	"os"
	"strings"
)

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiToken := os.Getenv("API_TOKEN")

		log.Printf("Loaded API_TOKEN: %q", apiToken)

		if apiToken == "" {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Server misconfiguration: No API_TOKEN set.",
			})
			return
		}

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
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Invalid Authorization header format",
			})
			return
		}

		receivedToken := parts[1]

		if receivedToken != apiToken {
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
