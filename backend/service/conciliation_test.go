package service

import (
	"errors"
	"math"
	"testing"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/models"
)

// --- in-memory SheetRepository adapter ---

type memRepo struct {
	sheets   map[string][][]interface{}
	appended map[string][][]interface{}
	cleared  map[string][]int
	written  []writtenCell
}

type writtenCell struct {
	sheet string
	row   int
	col   int
	value string
}

func newMemRepo(sheets map[string][][]interface{}) *memRepo {
	return &memRepo{
		sheets:   sheets,
		appended: make(map[string][][]interface{}),
		cleared:  make(map[string][]int),
	}
}

func (m *memRepo) FetchRows(sheet string) ([][]interface{}, error) {
	return m.sheets[sheet], nil
}

func (m *memRepo) WriteCell(sheet string, rowIdx, colIdx int, value string) error {
	m.written = append(m.written, writtenCell{sheet, rowIdx, colIdx, value})
	return nil
}

func (m *memRepo) AppendRow(sheet string, values []interface{}) error {
	m.appended[sheet] = append(m.appended[sheet], values)
	return nil
}

func (m *memRepo) ClearRow(sheet string, rowIdx int) error {
	m.cleared[sheet] = append(m.cleared[sheet], rowIdx)
	return nil
}

// --- Parser tests ---

var p Parser

func TestParseFloat(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected float64
	}{
		{"1234.56", 1234.56},
		{"1.234,56", 1234.56},
		{"R$ 1.234,56", 1234.56},
		{"R$50,00", 50.0},
		{"-50,00", -50.0},
		{"0", 0},
		{"", 0},
		{nil, 0},
		{1234.56, 1234.56},
		{"abc", 0},
	}

	for _, c := range cases {
		got := p.parseFloat(c.input)
		if math.Abs(got-c.expected) > 0.001 {
			t.Errorf("parseFloat(%v) = %v, want %v", c.input, got, c.expected)
		}
	}
}

func TestParseBool(t *testing.T) {
	cases := []struct {
		input    interface{}
		expected bool
	}{
		{"sim", true},
		{"SIM", true},
		{"Sim", true},
		{"yes", true},
		{"true", true},
		{"não", false},
		{"no", false},
		{"false", false},
		{"", false},
		{nil, false},
	}

	for _, c := range cases {
		got := p.parseBool(c.input)
		if got != c.expected {
			t.Errorf("parseBool(%v) = %v, want %v", c.input, got, c.expected)
		}
	}
}

func TestIsPending(t *testing.T) {
	cases := []struct {
		desc     string
		t        models.Transaction
		expected bool
	}{
		{
			desc:     "recorrente sem IdParcela",
			t:        models.Transaction{Recorrente: true, IdParcela: ""},
			expected: true,
		},
		{
			desc:     "recorrente com IdParcela preenchido",
			t:        models.Transaction{Recorrente: true, IdParcela: "abc-123"},
			expected: false,
		},
		{
			desc:     "nao recorrente sem IdParcela",
			t:        models.Transaction{Recorrente: false, IdParcela: ""},
			expected: false,
		},
		{
			desc:     "IdParcela com prefixo synthetic",
			t:        models.Transaction{Recorrente: false, IdParcela: "synthetic-456"},
			expected: true,
		},
		{
			desc:     "IdParcela synthetic com Recorrente true",
			t:        models.Transaction{Recorrente: true, IdParcela: "synthetic-789"},
			expected: true,
		},
		{
			desc:     "IdParcela com espacos e prefixo synthetic",
			t:        models.Transaction{Recorrente: false, IdParcela: "  synthetic-1"},
			expected: true,
		},
	}

	for _, c := range cases {
		got := p.IsPending(c.t)
		if got != c.expected {
			t.Errorf("[%s] IsPending() = %v, want %v", c.desc, got, c.expected)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	cases := []struct {
		desc     string
		row      []interface{}
		expected bool
	}{
		{"slice vazio", []interface{}{}, true},
		{"apenas strings vazias", []interface{}{"", ""}, true},
		{"apenas espacos", []interface{}{"   ", "  "}, true},
		{"com conteudo", []interface{}{"", "Alice"}, false},
		{"conteudo no primeiro campo", []interface{}{"valor", ""}, false},
	}

	for _, c := range cases {
		got := p.IsEmpty(c.row)
		if got != c.expected {
			t.Errorf("[%s] IsEmpty() = %v, want %v", c.desc, got, c.expected)
		}
	}
}

func makeTransaction(dono, banco, conta string, valor float64) models.Transaction {
	return models.Transaction{Dono: dono, Banco: banco, Conta: conta, Valor: valor}
}

func TestIsMatch(t *testing.T) {
	base := makeTransaction("Alice", "BancoBR", "Corrente", 100.0)

	cases := []struct {
		desc     string
		dif      models.Transaction
		es       models.Transaction
		expected bool
	}{
		{
			desc:     "match exato",
			dif:      base,
			es:       makeTransaction("Alice", "BancoBR", "Corrente", 100.0),
			expected: true,
		},
		{
			desc:     "diferenca de valor abaixo do limiar (4.99)",
			dif:      base,
			es:       makeTransaction("Alice", "BancoBR", "Corrente", 104.99),
			expected: true,
		},
		{
			desc:     "diferenca de valor no limiar exato (5.00) - deve falhar",
			dif:      base,
			es:       makeTransaction("Alice", "BancoBR", "Corrente", 105.0),
			expected: false,
		},
		{
			desc:     "diferenca de valor acima do limiar",
			dif:      base,
			es:       makeTransaction("Alice", "BancoBR", "Corrente", 106.0),
			expected: false,
		},
		{
			desc:     "dono diferente",
			dif:      base,
			es:       makeTransaction("Bob", "BancoBR", "Corrente", 100.0),
			expected: false,
		},
		{
			desc:     "banco diferente",
			dif:      base,
			es:       makeTransaction("Alice", "OutroBanco", "Corrente", 100.0),
			expected: false,
		},
		{
			desc:     "conta diferente",
			dif:      base,
			es:       makeTransaction("Alice", "BancoBR", "Poupanca", 100.0),
			expected: false,
		},
		{
			desc:     "valor negativo com diferenca aceitavel",
			dif:      makeTransaction("Alice", "BancoBR", "Corrente", -100.0),
			es:       makeTransaction("Alice", "BancoBR", "Corrente", -102.0),
			expected: true,
		},
	}

	for _, c := range cases {
		got := isMatch(c.dif, c.es)
		if got != c.expected {
			t.Errorf("[%s] isMatch() = %v, want %v", c.desc, got, c.expected)
		}
	}
}

// --- service-level tests using memRepo (sem rede) ---

func makeRow(dono, banco, conta, valor, idParcela, recorrente string) []interface{} {
	row := make([]interface{}, 10)
	row[models.ColumnDono] = dono
	row[models.ColumnBanco] = banco
	row[models.ColumnConta] = conta
	row[models.ColumnValor] = valor
	row[models.ColumnIdParcela] = idParcela
	row[models.ColumnRecorrente] = recorrente
	return row
}

func newTestLogic(t *testing.T, sheetData map[string][][]interface{}) *Logic {
	t.Helper()
	cfg := config.Config{
		SheetDIF: "DIF",
		SheetES:  "ES",
		SheetREJ: "REJ",
		SheetHOM: "HOM",
	}
	return NewLogic(newMemRepo(sheetData), cfg)
}

func TestAccept_WritesIdParcelaToES(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-42", "sim")
	esRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header, esRow},
	})
	logic := newTestLogic(t, repo.sheets)
	logic.repo = repo

	if err := logic.Accept(1, []int{1}); err != nil {
		t.Fatalf("Accept() error: %v", err)
	}

	if len(repo.written) != 1 {
		t.Fatalf("expected 1 WriteCell call, got %d", len(repo.written))
	}
	w := repo.written[0]
	if w.sheet != "ES" || w.row != 1 || w.col != models.ColumnIdParcela || w.value != "parcela-42" {
		t.Errorf("unexpected WriteCell: %+v", w)
	}
}

func TestReject_AppendsToREJAndClearsDIF(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Bob", "BankX", "Corrente", "50.00", "parcela-99", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"REJ": {header},
	})
	logic := newTestLogic(t, repo.sheets)
	logic.repo = repo

	if err := logic.Reject(1); err != nil {
		t.Fatalf("Reject() error: %v", err)
	}

	if len(repo.appended["REJ"]) != 1 {
		t.Fatalf("expected 1 row appended to REJ, got %d", len(repo.appended["REJ"]))
	}
	if len(repo.cleared["DIF"]) != 1 || repo.cleared["DIF"][0] != 1 {
		t.Errorf("expected DIF row 1 cleared, got %v", repo.cleared["DIF"])
	}
}

func TestListNonRecurringDIF_FiltersCorrectly(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	recurring := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	nonRecurring := makeRow("Bob", "BankX", "Poupanca", "200.00", "", "não")
	empty := []interface{}{}

	logic := newTestLogic(t, map[string][][]interface{}{
		"DIF": {header, recurring, nonRecurring, empty},
	})
	items, err := logic.ListNonRecurringDIF()
	if err != nil {
		t.Fatalf("ListNonRecurringDIF() error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 non-recurring item, got %d", len(items))
	}
	if items[0].Dono != "Bob" {
		t.Errorf("expected Dono=Bob, got %s", items[0].Dono)
	}
}

func newTestLogicWithRepo(t *testing.T, repo *memRepo) *Logic {
	t.Helper()
	cfg := config.Config{SheetDIF: "DIF", SheetES: "ES", SheetREJ: "REJ", SheetHOM: "HOM"}
	return NewLogic(repo, cfg)
}

// --- GetConciliations ---

func TestGetConciliations_CountsCandidates(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	esRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header, esRow},
	})
	results, err := newTestLogicWithRepo(t, repo).GetConciliations()
	if err != nil {
		t.Fatalf("GetConciliations() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].CandidateCount != 1 {
		t.Errorf("expected CandidateCount=1, got %d", results[0].CandidateCount)
	}
}

func TestGetConciliations_SkipsNonRecurringDIF(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "não")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header},
	})
	results, err := newTestLogicWithRepo(t, repo).GetConciliations()
	if err != nil {
		t.Fatalf("GetConciliations() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- GetConciliationDetails ---

func TestGetConciliationDetails_ReturnsCandidates(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	esRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header, esRow},
	})
	result, err := newTestLogicWithRepo(t, repo).GetConciliationDetails(1)
	if err != nil {
		t.Fatalf("GetConciliationDetails() error: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Errorf("expected 1 candidate, got %d", len(result.Candidates))
	}
}

func TestGetConciliationDetails_OutOfBounds(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header}})
	_, err := newTestLogicWithRepo(t, repo).GetConciliationDetails(5)
	if err == nil {
		t.Error("expected error for out-of-bounds index")
	}
}

func TestGetConciliationDetails_NonRecurring(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "não")
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header, difRow}})
	_, err := newTestLogicWithRepo(t, repo).GetConciliationDetails(1)
	if err == nil {
		t.Error("expected error for non-recurring DIF row")
	}
}

// --- Accept error paths ---

func TestAccept_OutOfBounds(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header}})
	err := newTestLogicWithRepo(t, repo).Accept(5, []int{1})
	if err == nil {
		t.Error("expected error for out-of-bounds index")
	}
}

func TestAccept_EmptyIdParcela(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "", "sim")
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header, difRow}})
	err := newTestLogicWithRepo(t, repo).Accept(1, []int{1})
	if err == nil {
		t.Error("expected error for empty IdParcela")
	}
}

// --- Move tests ---

func TestMoveNonRecurringDifToES_MovesRow(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Bob", "BankX", "Poupanca", "200.00", "", "não")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header},
	})
	if err := newTestLogicWithRepo(t, repo).MoveNonRecurringDifToES(1); err != nil {
		t.Fatalf("MoveNonRecurringDifToES() error: %v", err)
	}
	if len(repo.appended["ES"]) != 1 {
		t.Errorf("expected row appended to ES, got %d", len(repo.appended["ES"]))
	}
	if len(repo.cleared["DIF"]) != 1 || repo.cleared["DIF"][0] != 1 {
		t.Errorf("expected DIF row 1 cleared, got %v", repo.cleared["DIF"])
	}
}

func TestMoveNonRecurringDifToES_RejectsRecurring(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header, difRow}})
	err := newTestLogicWithRepo(t, repo).MoveNonRecurringDifToES(1)
	if err == nil {
		t.Error("expected error for recurring DIF row")
	}
}

func TestMoveNonRecurringDifToREJ_MovesRow(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Bob", "BankX", "Poupanca", "200.00", "", "não")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"REJ": {header},
	})
	if err := newTestLogicWithRepo(t, repo).MoveNonRecurringDifToREJ(1); err != nil {
		t.Fatalf("MoveNonRecurringDifToREJ() error: %v", err)
	}
	if len(repo.appended["REJ"]) != 1 {
		t.Errorf("expected row appended to REJ, got %d", len(repo.appended["REJ"]))
	}
	if len(repo.cleared["DIF"]) != 1 {
		t.Errorf("expected DIF row cleared, got %v", repo.cleared["DIF"])
	}
}

func TestMoveAllNonRecurringDifToES_MovesAll(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	row1 := makeRow("Bob", "BankX", "Poupanca", "200.00", "", "não")
	row2 := makeRow("Carol", "BankY", "Corrente", "300.00", "", "não")
	recurring := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, row1, row2, recurring},
		"ES":  {header},
	})
	result, err := newTestLogicWithRepo(t, repo).MoveAllNonRecurringDifToES()
	if err != nil {
		t.Fatalf("MoveAllNonRecurringDifToES() error: %v", err)
	}
	if result.MovedToES != 2 {
		t.Errorf("expected MovedToES=2, got %d", result.MovedToES)
	}
	if len(repo.appended["ES"]) != 2 {
		t.Errorf("expected 2 rows appended to ES, got %d", len(repo.appended["ES"]))
	}
}

// --- UpdateDifCategory / UpdateDifDate ---

// A HOM tem a linha-alvo DESLOCADA: se o serviço usasse o índice da DIF, escreveria
// na linha errada (o bug do #21). Como endereça por IdParcela, acerta a linha certa.
func TestUpdateDifCategory_WritesCellByIdParcela(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	other := makeRow("Zed", "BancoX", "Corrente", "50.00", "outra-parcela", "não")
	target := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-7", "não")

	// alvo na linha 2 da HOM — um índice de DIF apontaria para a linha 1.
	repo := newMemRepo(map[string][][]interface{}{"HOM": {header, other, target}})
	if err := newTestLogicWithRepo(t, repo).UpdateDifCategory("parcela-7", "Alimentação"); err != nil {
		t.Fatalf("UpdateDifCategory() error: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatalf("expected 1 WriteCell call, got %d", len(repo.written))
	}
	w := repo.written[0]
	if w.sheet != "HOM" || w.row != 2 || w.col != models.ColumnCategoria || w.value != "Alimentação" {
		t.Errorf("unexpected WriteCell: %+v", w)
	}
}

func TestUpdateDifDate_WritesCellByIdParcela(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	other := makeRow("Zed", "BancoX", "Corrente", "50.00", "outra-parcela", "não")
	target := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-7", "não")

	repo := newMemRepo(map[string][][]interface{}{"HOM": {header, other, target}})
	if err := newTestLogicWithRepo(t, repo).UpdateDifDate("parcela-7", "2026-06-14"); err != nil {
		t.Fatalf("UpdateDifDate() error: %v", err)
	}
	if len(repo.written) != 1 {
		t.Fatalf("expected 1 WriteCell call, got %d", len(repo.written))
	}
	w := repo.written[0]
	if w.sheet != "HOM" || w.row != 2 || w.col != models.ColumnData || w.value != "2026-06-14" {
		t.Errorf("unexpected WriteCell: %+v", w)
	}
}

func TestUpdateDifCategory_NotInHOM(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	homRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-7", "não")
	repo := newMemRepo(map[string][][]interface{}{"HOM": {header, homRow}})

	err := newTestLogicWithRepo(t, repo).UpdateDifCategory("inexistente", "Alimentação")
	if !errors.Is(err, ErrTransactionNotInHOM) {
		t.Fatalf("expected ErrTransactionNotInHOM, got %v", err)
	}
	if len(repo.written) != 0 {
		t.Errorf("expected no WriteCell, got %d", len(repo.written))
	}
}

func TestUpdateDifDate_NotInHOM(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	homRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-7", "não")
	repo := newMemRepo(map[string][][]interface{}{"HOM": {header, homRow}})

	err := newTestLogicWithRepo(t, repo).UpdateDifDate("inexistente", "2026-06-14")
	if !errors.Is(err, ErrTransactionNotInHOM) {
		t.Fatalf("expected ErrTransactionNotInHOM, got %v", err)
	}
	if len(repo.written) != 0 {
		t.Errorf("expected no WriteCell, got %d", len(repo.written))
	}
}

func TestUpdateDifCategory_EmptyIdParcela(t *testing.T) {
	repo := newMemRepo(map[string][][]interface{}{})
	err := newTestLogicWithRepo(t, repo).UpdateDifCategory("  ", "Alimentação")
	if !errors.Is(err, ErrEmptyIdParcela) {
		t.Fatalf("expected ErrEmptyIdParcela, got %v", err)
	}
}

func TestUpdateDifDate_EmptyIdParcela(t *testing.T) {
	repo := newMemRepo(map[string][][]interface{}{})
	err := newTestLogicWithRepo(t, repo).UpdateDifDate("", "2026-06-14")
	if !errors.Is(err, ErrEmptyIdParcela) {
		t.Fatalf("expected ErrEmptyIdParcela, got %v", err)
	}
}

// --- Gaps restantes ---

func TestGetConciliations_SkipsEmptyDIFRow(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	emptyRow := []interface{}{}

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, emptyRow},
		"ES":  {header},
	})
	results, err := newTestLogicWithRepo(t, repo).GetConciliations()
	if err != nil {
		t.Fatalf("GetConciliations() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty DIF row, got %d", len(results))
	}
}

func TestGetConciliationDetails_NoMatchingCandidates(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	esRow := makeRow("Bob", "OutroBanco", "Poupanca", "999.00", "", "sim")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, difRow},
		"ES":  {header, esRow},
	})
	result, err := newTestLogicWithRepo(t, repo).GetConciliationDetails(1)
	if err != nil {
		t.Fatalf("GetConciliationDetails() error: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(result.Candidates))
	}
}

func TestReject_OutOfBounds(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header}})
	err := newTestLogicWithRepo(t, repo).Reject(5)
	if err == nil {
		t.Error("expected error for out-of-bounds index")
	}
}

func TestMoveNonRecurringDifToREJ_RejectsRecurring(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	difRow := makeRow("Alice", "BancoBR", "Corrente", "100.00", "p-1", "sim")
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header, difRow}})
	err := newTestLogicWithRepo(t, repo).MoveNonRecurringDifToREJ(1)
	if err == nil {
		t.Error("expected error for recurring DIF row")
	}
}

func TestMoveNonRecurringDifToREJ_OutOfBounds(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	repo := newMemRepo(map[string][][]interface{}{"DIF": {header}})
	err := newTestLogicWithRepo(t, repo).MoveNonRecurringDifToREJ(5)
	if err == nil {
		t.Error("expected error for out-of-bounds index")
	}
}

func TestMoveAllNonRecurringDifToES_SkipsEmptyRows(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	emptyRow := []interface{}{}
	nonRecurring := makeRow("Bob", "BankX", "Poupanca", "200.00", "", "não")

	repo := newMemRepo(map[string][][]interface{}{
		"DIF": {header, emptyRow, nonRecurring},
		"ES":  {header},
	})
	result, err := newTestLogicWithRepo(t, repo).MoveAllNonRecurringDifToES()
	if err != nil {
		t.Fatalf("MoveAllNonRecurringDifToES() error: %v", err)
	}
	if result.MovedToES != 1 {
		t.Errorf("expected MovedToES=1 (empty row skipped), got %d", result.MovedToES)
	}
}

// Uma linha vazia da HOM não deve casar com um IdParcela não-vazio nem quebrar a busca.
func TestUpdateDifCategory_SkipsEmptyHOMRow(t *testing.T) {
	header := []interface{}{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	emptyRow := []interface{}{}
	target := makeRow("Alice", "BancoBR", "Corrente", "100.00", "parcela-7", "não")
	repo := newMemRepo(map[string][][]interface{}{"HOM": {header, emptyRow, target}})

	if err := newTestLogicWithRepo(t, repo).UpdateDifCategory("parcela-7", "Alimentação"); err != nil {
		t.Fatalf("UpdateDifCategory() error: %v", err)
	}
	if len(repo.written) != 1 || repo.written[0].row != 2 {
		t.Errorf("expected WriteCell on row 2, got %+v", repo.written)
	}
}
