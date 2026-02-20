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

	// Public Routes
	// Use path-based patterns for broader compatibility across deployments.
	mux.HandleFunc("/api/login", h.Login)
	mux.HandleFunc("/api/logout", h.Logout)
	mux.HandleFunc("/api/auth/verify", h.Verify)

	// Protected Routes
	protectedMux := http.NewServeMux()
	protectedMux.HandleFunc("/api/conciliations", h.GetConciliations)

	protectedMux.HandleFunc("/api/conciliations/", func(w http.ResponseWriter, r *http.Request) {
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
