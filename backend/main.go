package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"slices"
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

	requiredEnvVars := []string{
		"SHEET_SPREADSHEET_ID",
		"ADMIN_USER",
		"ADMIN_PASS",
		"JWT_SECRET",
		"SHEET_ES",
		"SHEET_DIF",
		"SHEET_REJ",
	}

	missingEnvVars := collectMissingEnvVars(requiredEnvVars)
	if len(missingEnvVars) > 0 {
		log.Printf("missing or empty required env vars: %s", strings.Join(missingEnvVars, ", "))
		log.Fatal("startup aborted due to missing required env vars")
	}

	spreadsheetID := os.Getenv("SHEET_SPREADSHEET_ID")

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

func collectMissingEnvVars(names []string) []string {
	missing := make([]string, 0)
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		value, exists := os.LookupEnv(name)
		if !exists || strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}

	slices.Sort(missing)
	return missing
}
