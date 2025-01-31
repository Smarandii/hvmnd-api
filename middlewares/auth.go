package middleware

import (
	"hvmnd/api/models"
	"hvmnd/api/utils"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

// parseClientIP extracts the IP address portion from the remote address
func parseClientIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", err
	}
	return host, nil
}

// isIPWhitelisted checks if clientIP is in a comma-separated list of IPs
func isIPWhitelisted(clientIP, whitelist string) bool {
	if whitelist == "" {
		// If no whitelist is set, you might consider all IPs as valid
		// or treat it as no IPs are allowed. Adjust as needed.
		return false
	}
	for _, wIP := range strings.Split(whitelist, ",") {
		trimmed := strings.TrimSpace(wIP)
		if trimmed == clientIP {
			return true
		}
	}
	return false
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiToken := os.Getenv("API_TOKEN")
		nodesApiToken := os.Getenv("NODES_API_TOKEN")
		whitelistIPs := os.Getenv("COMMA_SEPARATED_WHITELIST_IPS")

		log.Printf("Loaded API_TOKEN: %q", apiToken)
		log.Printf("Loaded NODES_API_TOKEN: %q", nodesApiToken)
		log.Printf("Loaded Whitelist IPs: %q", whitelistIPs)

		if apiToken == "" {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Server misconfiguration: No API_TOKEN set.",
			})
			return
		}

		if nodesApiToken == "" {
			utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Server misconfiguration: No NODES_API_TOKEN set.",
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
		log.Printf("nodesApiToken from env: %q", nodesApiToken)
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
		// Decide which token we are dealing with (apiToken or nodesApiToken)
		if receivedToken != apiToken && receivedToken != nodesApiToken {
			utils.WriteJSONResponse(w, http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "Invalid or missing token",
			})
			return
		}

		// If the received token is the NODES_API_TOKEN, verify the request IP is whitelisted
		if receivedToken == nodesApiToken {
			// Parse the remote IP (excluding the port)
			clientIP, err := parseClientIP(r.RemoteAddr)
			if err != nil {
				log.Printf("Could not parse client IP: %v", err)
				utils.WriteJSONResponse(w, http.StatusInternalServerError, models.APIResponse{
					Success: false,
					Message: "Could not parse client IP",
					Error:   err.Error(),
				})
				return
			}

			// Check if clientIP is in the list of whitelisted IPs
			if !isIPWhitelisted(clientIP, whitelistIPs) {
				log.Printf("Client IP %s is not whitelisted", clientIP)
				utils.WriteJSONResponse(w, http.StatusForbidden, models.APIResponse{
					Success: false,
					Message: "IP not whitelisted",
				})
				return
			}

			// IP is whitelisted, proceed
			log.Printf("Client IP %s is whitelisted", clientIP)
		}

		// Token is valid, proceed
		next.ServeHTTP(w, r)
	})
}
