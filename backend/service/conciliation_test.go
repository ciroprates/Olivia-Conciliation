package service

import (
	"math"
	"testing"

	"olivia-conciliation/backend/models"
)

// --- parseFloat ---

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
		{1234.56, 1234.56},   // numeric input via interface{}
		{"abc", 0},
	}

	for _, c := range cases {
		got := parseFloat(c.input)
		if math.Abs(got-c.expected) > 0.001 {
			t.Errorf("parseFloat(%v) = %v, want %v", c.input, got, c.expected)
		}
	}
}

// --- parseBool ---

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
		got := parseBool(c.input)
		if got != c.expected {
			t.Errorf("parseBool(%v) = %v, want %v", c.input, got, c.expected)
		}
	}
}

// --- isPendingES ---

func TestIsPendingES(t *testing.T) {
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
		got := isPendingES(c.t)
		if got != c.expected {
			t.Errorf("[%s] isPendingES() = %v, want %v", c.desc, got, c.expected)
		}
	}
}

// --- isMatch ---

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

// --- isEmptyRow ---

func TestIsEmptyRow(t *testing.T) {
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
		got := isEmptyRow(c.row)
		if got != c.expected {
			t.Errorf("[%s] isEmptyRow() = %v, want %v", c.desc, got, c.expected)
		}
	}
}
