package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/service"
)

func newTestHandler() *Handler {
	cfg := config.Config{
		AdminUser: "admin",
		AdminPass: "secret",
		JWTSecret: "test-jwt-secret",
		AppOrigin: "http://localhost:3001",
	}
	return NewHandler(&service.Logic{}, cfg)
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
