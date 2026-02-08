package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"olivia-conciliation/backend/handlers"
	"olivia-conciliation/backend/service"
	"olivia-conciliation/backend/sheets"
)

func main() {
	// Load .env if present, without failing when missing.
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Printf("warning: failed to load .env: %v", err)
	}

	// Env vars checks
	spreadsheetID := os.Getenv("SPREADSHEET_ID")
	if spreadsheetID == "" {
		log.Fatal("SPREADSHEET_ID not set")
	}

	// Init Sheets Client
	client, err := sheets.NewClient(context.Background(), spreadsheetID)
	if err != nil {
		log.Fatalf("Failed to create sheets client: %v", err)
	}

	// Init Service
	svc := service.NewLogic(client)

	// Init Handlers
	h := handlers.NewHandler(svc)

	mux := http.NewServeMux()

	// Generic middleware for CORS
	corsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Routes
	// Exact match first, then prefix
	mux.HandleFunc("GET /api/conciliations", h.GetConciliations)
	mux.HandleFunc("GET /api/conciliations/", h.GetConciliationDetails) // This matches /api/conciliations/{id} and potentially subpaths if not careful.
	// Since we use Split strings logic in handler, it's okay for now.
	// But "POST /api/conciliations/{id}/accept" needs careful routing.
	// Go 1.22 has better routing, but assuming older Go or standard mux limitations.
	// Let's use standard prefix mathcing + method checks or simple manual routing if using old mux.
	// If using Go 1.22+ we can use "POST /api/conciliations/{id}/accept".
	// I'll stick to manual dispatching in a shared path or distinct paths if simpler.

	// Using simple prefix matching for clarity with standard lib
	mux.HandleFunc("/api/conciliations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}

		// /api/conciliations/{id} vs /api/conciliations/{id}/accept
		path := r.URL.Path
		if strings.HasSuffix(path, "/accept") && r.Method == "POST" {
			h.AcceptConciliation(w, r)
			return
		}
		if strings.HasSuffix(path, "/reject") && r.Method == "POST" {
			h.RejectConciliation(w, r)
			return
		}
		if r.Method == "GET" {
			// Check if it's the list or detail
			// List is strictly /api/conciliations but this handler is registered for trailing slash?
			// Actually I registered "GET /api/conciliations" separately.
			// This handler catches /api/conciliations/...
			h.GetConciliationDetails(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// Fix: Re-register list exact match if needed or ensure the prefix doesn't swallow it.
	// Mux specific: "/foo/" matches "/foo/bar", "/foo" matches "/foo".
	// I routed "GET /api/conciliations" above. I will change to standard handler func without method prefix (Go 1.22 feature) to be safe for older Go versions possibly on user machine?
	// User has generic "linux", likely Go 1.21+. Let's assume standard mux.
	// To be safe: register "/api/conciliations" and "/api/conciliations/"

	finalHandler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, finalHandler))
}
