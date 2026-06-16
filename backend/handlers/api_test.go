package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/models"
	"olivia-conciliation/backend/service"
)

// fakeRepo implementa service.SheetRepository com dados em memória.
type fakeRepo struct {
	sheets   map[string][][]interface{}
	appended map[string][][]interface{}
	cleared  map[string][]int
	written  []struct{ sheet string; row, col int; value string }
}

func newFakeRepo(sheets map[string][][]interface{}) *fakeRepo {
	return &fakeRepo{
		sheets:   sheets,
		appended: make(map[string][][]interface{}),
		cleared:  make(map[string][]int),
	}
}

func (f *fakeRepo) FetchRows(sheet string) ([][]interface{}, error) {
	return f.sheets[sheet], nil
}
func (f *fakeRepo) WriteCell(sheet string, row, col int, value string) error {
	f.written = append(f.written, struct{ sheet string; row, col int; value string }{sheet, row, col, value})
	return nil
}
func (f *fakeRepo) AppendRow(sheet string, values []interface{}) error {
	f.appended[sheet] = append(f.appended[sheet], values)
	return nil
}
func (f *fakeRepo) ClearRow(sheet string, row int) error {
	f.cleared[sheet] = append(f.cleared[sheet], row)
	return nil
}

func newAPIHandler(repo *fakeRepo) *Handler {
	cfg := config.Config{SheetDIF: "DIF", SheetES: "ES", SheetREJ: "REJ", SheetHOM: "HOM"}
	svc := service.NewLogic(repo, cfg)
	return NewHandler(svc, cfg)
}

var apiHeader = []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}

func apiRow(dono, banco, conta, valor, idParcela, recorrente string) []interface{} {
	row := make([]interface{}, 10)
	row[models.ColumnDono] = dono
	row[models.ColumnBanco] = banco
	row[models.ColumnConta] = conta
	row[models.ColumnValor] = valor
	row[models.ColumnIdParcela] = idParcela
	row[models.ColumnRecorrente] = recorrente
	return row
}

// --- API handler tests ---

func TestGetConciliations_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")},
		"ES":  {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations", nil)
	w := httptest.NewRecorder()

	h.GetConciliations(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result []models.PendingConciliationSummary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestGetConciliations_MethodNotAllowed(t *testing.T) {
	h := newAPIHandler(newFakeRepo(nil))
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations", nil)
	w := httptest.NewRecorder()
	h.GetConciliations(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestGetConciliationDetails_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")},
		"ES":  {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations/1", nil)
	w := httptest.NewRecorder()

	h.GetConciliationDetails(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetConciliationDetails_InvalidPath(t *testing.T) {
	h := newAPIHandler(newFakeRepo(nil))
	r := httptest.NewRequest(http.MethodGet, "/api/conciliations/abc", nil)
	w := httptest.NewRecorder()
	h.GetConciliationDetails(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAcceptConciliation_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")},
		"ES":  {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")},
	})
	h := newAPIHandler(repo)
	body := strings.NewReader(`{"esRowIndices":[1]}`)
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", body)
	w := httptest.NewRecorder()

	h.AcceptConciliation(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if len(repo.written) != 1 {
		t.Errorf("expected WriteCell called once, got %d", len(repo.written))
	}
}

func TestAcceptConciliation_InvalidJSON(t *testing.T) {
	h := newAPIHandler(newFakeRepo(nil))
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", strings.NewReader("bad"))
	w := httptest.NewRecorder()
	h.AcceptConciliation(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAcceptConciliation_InvalidPath(t *testing.T) {
	h := newAPIHandler(newFakeRepo(nil))
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/abc/accept", strings.NewReader(`{"esRowIndices":[]}`))
	w := httptest.NewRecorder()
	h.AcceptConciliation(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestRejectConciliation_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")},
		"REJ": {apiHeader},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/reject", nil)
	w := httptest.NewRecorder()

	h.RejectConciliation(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListNonRecurringDif_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Bob", "BankX", "Poupanca", "200.00", "", "não")},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodGet, "/api/dif/non-recurring", nil)
	w := httptest.NewRecorder()

	h.ListNonRecurringDif(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var result []models.NonRecurringDifSummary
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestMoveNonRecurringDifToES_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Bob", "BankX", "Poupanca", "200.00", "", "não")},
		"ES":  {apiHeader},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodPost, "/api/dif/non-recurring/1/move-to-es", nil)
	w := httptest.NewRecorder()

	h.MoveNonRecurringDifToES(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMoveNonRecurringDifToREJ_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader, apiRow("Bob", "BankX", "Poupanca", "200.00", "", "não")},
		"REJ": {apiHeader},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodPost, "/api/dif/non-recurring/1/move-to-rej", nil)
	w := httptest.NewRecorder()

	h.MoveNonRecurringDifToREJ(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMoveAllNonRecurringDifToES_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"DIF": {apiHeader,
			apiRow("Bob", "BankX", "Poupanca", "200.00", "", "não"),
			apiRow("Carol", "BankY", "Corrente", "300.00", "", "não"),
		},
		"ES": {apiHeader},
	})
	h := newAPIHandler(repo)
	r := httptest.NewRequest(http.MethodPost, "/api/dif/non-recurring/move-all-to-es", nil)
	w := httptest.NewRecorder()

	h.MoveAllNonRecurringDifToES(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result models.NonRecurringBulkActionResult
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.MovedToES != 2 {
		t.Errorf("expected MovedToES=2, got %d", result.MovedToES)
	}
}

func TestUpdateNonRecurringDifCategory_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{
		"HOM": {apiHeader, apiRow("Bob", "BankX", "Poupanca", "200.00", "", "não")},
	})
	h := newAPIHandler(repo)
	body := strings.NewReader(`{"categoria":"Alimentação"}`)
	r := httptest.NewRequest(http.MethodPatch, "/api/dif/non-recurring/1/category", body)
	w := httptest.NewRecorder()

	h.UpdateNonRecurringDifCategory(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateNonRecurringDifDate_Returns200(t *testing.T) {
	repo := newFakeRepo(map[string][][]interface{}{})
	h := newAPIHandler(repo)
	body := strings.NewReader(`{"data":"2026-06-15"}`)
	r := httptest.NewRequest(http.MethodPatch, "/api/dif/non-recurring/1/date", body)
	w := httptest.NewRecorder()

	h.UpdateNonRecurringDifDate(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// --- validateTrustedOrigin: path do Referer ---

func TestAuthMiddleware_ValidToken_POST_ValidReferer_PassesThrough(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.AddCookie(&http.Cookie{Name: "olivia_csrf", Value: "tok123"})
	r.Header.Set("Referer", testOrigin+"/queue")
	r.Header.Set("X-CSRF-Token", "tok123")
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_ValidToken_POST_InvalidReferer_Returns403(t *testing.T) {
	h := newTestHandler()
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequest(http.MethodPost, "/api/conciliations/1/accept", nil)
	r.AddCookie(&http.Cookie{Name: "olivia_session", Value: makeValidToken(testSecret)})
	r.AddCookie(&http.Cookie{Name: "olivia_csrf", Value: "tok123"})
	r.Header.Set("Referer", "http://evil.com/page")
	r.Header.Set("X-CSRF-Token", "tok123")
	w := httptest.NewRecorder()

	h.AuthMiddleware(next).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

// --- ExtractPathID ---

func TestExtractPathID(t *testing.T) {
	cases := []struct {
		path    string
		depth   int
		wantID  int
		wantErr bool
	}{
		{"/api/conciliations/42", 1, 42, false},
		{"/api/conciliations/42/accept", 2, 42, false},
		{"/api/dif/non-recurring/7/move-to-es", 2, 7, false},
		{"/short", 2, 0, true},
		{"/api/conciliations/abc", 1, 0, true},
	}

	for _, c := range cases {
		got, err := extractPathID(c.path, c.depth)
		if c.wantErr {
			if err == nil {
				t.Errorf("extractPathID(%q, %d): expected error, got %d", c.path, c.depth, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("extractPathID(%q, %d): unexpected error: %v", c.path, c.depth, err)
			continue
		}
		if got != c.wantID {
			t.Errorf("extractPathID(%q, %d) = %d, want %d", c.path, c.depth, got, c.wantID)
		}
	}
}
