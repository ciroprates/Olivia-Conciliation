package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"olivia-conciliation/backend/handlers"
	"olivia-conciliation/backend/service"
	"olivia-conciliation/backend/sheets"

	"github.com/joho/godotenv"
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

	// Public Routes
	mux.HandleFunc("POST /api/login", h.Login)

	// Protected Routes
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("GET /api/auth/verify", h.Verify)
	protectedMux.HandleFunc("GET /api/conciliations", h.GetConciliations)

	protectedMux.HandleFunc("/api/conciliations/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			return
		}

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
			h.GetConciliationDetails(w, r)
			return
		}
		http.NotFound(w, r)
	})

	// Mount protected routes with AuthMiddleware
	mux.Handle("/api/", h.AuthMiddleware(protectedMux))

	finalHandler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, finalHandler))
}
