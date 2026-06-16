package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/service"
)

const testSecret = "test-jwt-secret"
const testOrigin = "http://localhost:3001"

func newTestHandler() *Handler {
	cfg := config.Config{
		AdminUser: "admin",
		AdminPass: "secret",
		JWTSecret: testSecret,
		AppOrigin: testOrigin,
	}
	return NewHandler(&service.Logic{}, cfg)
}

func makeValidToken(secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": "admin",
		"exp":  time.Now().Add(time.Hour).Unix(),
	})
	str, _ := token.SignedString([]byte(secret))
	return str
}

func makeExpiredToken(secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user": "admin",
		"exp":  time.Now().Add(-time.Hour).Unix(),
	})
	str, _ := token.SignedString([]byte(secret))
	return str
}

func TestLogin_Success(t *testing.T) {
	h := newTestHandler()
	body := strings.NewReader(`{"username":"admin","password":"secret"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", body)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	cookies := w.Result().Cookies()
	var hasSession, hasCSRF bool
	for _, c := range cookies {
		if c.Name == "olivia_session" {
			hasSession = true
		}
		if c.Name == "olivia_csrf" {
			hasCSRF = true
		}
	}
	if !hasSession {
		t.Error("expected olivia_session cookie")
	}
	if !hasCSRF {
		t.Error("expected olivia_csrf cookie")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newTestHandler()
	body := strings.NewReader(`{"username":"admin","password":"wrong"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", body)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestLogin_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/login", nil)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestLogin_InvalidJSON(t *testing.T) {
	h := newTestHandler()
	body := strings.NewReader(`not json`)
	r := httptest.NewRequest(http.MethodPost, "/api/login", body)
	w := httptest.NewRecorder()

	h.Login(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestLogout_ClearsCookies(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodPost, "/api/logout", nil)
	w := httptest.NewRecorder()

	h.Logout(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "olivia_session" || c.Name == "olivia_csrf" {
			if c.MaxAge != -1 {
				t.Errorf("expected cookie %s to be cleared (MaxAge=-1), got MaxAge=%d", c.Name, c.MaxAge)
			}
		}
	}
}

// --- AuthMiddleware ---

func TestAuthMiddleware_NoToken_Returns401(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations", nil)
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_GET_PassesThrough(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken_Returns401(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeExpiredToken(testSecret)})
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_POST_WithCSRF_PassesThrough(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.AddCookie(&http.Cookie{Name: "olivia_csrf", Value: "tok123"})
	r.Header.Set("Origin", testOrigin)
	r.Header.Set("X-CSRF-Token", "tok123")
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_POST_MissingCSRF_Returns403(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.Header.Set("Origin", testOrigin)
	// sem olivia_csrf cookie e sem X-CSRF-Token
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_POST_InvalidOrigin_Returns403(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.AddCookie(&http.Cookie{Name: "olivia_csrf", Value: "tok123"})
	r.Header.Set("Origin", "http://evil.com")
	r.Header.Set("X-CSRF-Token", "tok123")
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_POST_CSRFMismatch_Returns403(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.AddCookie(&http.Cookie{Name: "olivia_csrf", Value: "tok-cookie"})
	r.Header.Set("Origin", testOrigin)
	r.Header.Set("X-CSRF-Token", "tok-different")
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- Verify ---

func TestVerify_WithValidToken_Returns200(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/verify", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	w := httptest.NewRecorder()

	h.Verify(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestVerify_WithoutToken_Returns401(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/verify", nil)
	w := httptest.NewRecorder()

	h.Verify(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestVerify_MethodNotAllowed(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodPost, "/api/verify", nil)
	w := httptest.NewRecorder()

	h.Verify(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestVerify_WithBearerToken_Returns200(t *testing.T) {
	h := newTestHandler()
	r := httptest.NewRequest(http.MethodGet, "/api/verify", nil)
	r.Header.Set("Authorization", "Bearer "+makeValidToken(testSecret))
	w := httptest.NewRecorder()

	h.Verify(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}
