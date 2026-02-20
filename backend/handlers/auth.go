package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Authenticated bool `json:"authenticated"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")
	jwtSecret := os.Getenv("JWT_SECRET")

	if adminUser == "" || adminPass == "" || jwtSecret == "" {
		http.Error(w, "Auth configuration missing", http.StatusInternalServerError)
		return
	}

	if req.Username != adminUser || req.Password != adminPass {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	csrfToken, err := generateCSRFToken()
	if err != nil {
		http.Error(w, "Failed to generate CSRF token", http.StatusInternalServerError)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": req.Username,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	setAuthCookies(w, tokenString, csrfToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(LoginResponse{Authenticated: true})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clearAuthCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := parseRequestToken(r); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if requiresCSRF(r.Method) {
			if err := validateCSRFFromRequest(r); err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if _, err := parseRequestToken(r); err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	originMethod := strings.ToUpper(strings.TrimSpace(r.Header.Get("X-Original-Method")))
	if originMethod == "" {
		originMethod = r.Method
	}

	if requiresCSRF(originMethod) {
		if err := validateCSRFFromRequest(r); err != nil {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

func parseRequestToken(r *http.Request) (*jwt.Token, error) {
	tokenString := extractBearerToken(r.Header.Get("Authorization"))
	if tokenString == "" {
		cookie, err := r.Cookie("olivia_session")
		if err == nil {
			tokenString = cookie.Value
		}
	}
	if tokenString == "" {
		return nil, errors.New("missing token")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("missing jwt secret")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return token, nil
}

func extractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}
	return parts[1]
}

func requiresCSRF(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func validateCSRFFromRequest(r *http.Request) error {
	if err := validateTrustedOrigin(r); err != nil {
		return err
	}

	cookie, err := r.Cookie("olivia_csrf")
	if err != nil || cookie.Value == "" {
		return errors.New("missing csrf cookie")
	}

	headerToken := strings.TrimSpace(r.Header.Get("X-CSRF-Token"))
	if headerToken == "" {
		return errors.New("missing csrf header")
	}

	if subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(headerToken)) != 1 {
		return errors.New("csrf mismatch")
	}

	return nil
}

func validateTrustedOrigin(r *http.Request) error {
	allowedOrigin := strings.TrimSpace(os.Getenv("APP_ORIGIN"))
	if allowedOrigin == "" {
		allowedOrigin = "https://console.olivinha.site"
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin != "" {
		if origin != allowedOrigin {
			return errors.New("invalid origin")
		}
		return nil
	}

	referer := strings.TrimSpace(r.Header.Get("Referer"))
	if referer != "" && !strings.HasPrefix(referer, allowedOrigin+"/") {
		return errors.New("invalid referer")
	}

	return nil
}

func generateCSRFToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func setAuthCookies(w http.ResponseWriter, sessionToken, csrfToken string) {
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	secureCookies := os.Getenv("COOKIE_SECURE") != "false"
	maxAge := 24 * 60 * 60
	expiresAt := time.Now().Add(24 * time.Hour)

	sessionCookie := &http.Cookie{
		Name:     "olivia_session",
		Value:    sessionToken,
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  expiresAt,
	}
	http.SetCookie(w, sessionCookie)

	csrfCookie := &http.Cookie{
		Name:     "olivia_csrf",
		Value:    csrfToken,
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: false,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
		Expires:  expiresAt,
	}
	http.SetCookie(w, csrfCookie)
}

func clearAuthCookies(w http.ResponseWriter) {
	cookieDomain := os.Getenv("COOKIE_DOMAIN")
	secureCookies := os.Getenv("COOKIE_SECURE") != "false"

	http.SetCookie(w, &http.Cookie{
		Name:     "olivia_session",
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: true,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "olivia_csrf",
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HttpOnly: false,
		Secure:   secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}
