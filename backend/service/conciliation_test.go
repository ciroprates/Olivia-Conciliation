package service

import (
	"math"
	"testing"

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
	t.Setenv("SHEET_DIF", "DIF")
	t.Setenv("SHEET_ES", "ES")
	t.Setenv("SHEET_REJ", "REJ")
	t.Setenv("SHEET_HOM", "HOM")
	return NewLogic(newMemRepo(sheetData))
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
