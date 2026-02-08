package models

// Column indices (A=1, so index 0 would be A-1, but Sheets API GridRange sometimes uses 0-based.
// However, when parsing a row slice []interface{}, index 0 is Column A.
const (
	ColumnData      = 1 // B
	ColumnDescricao = 2 // C
	ColumnValor     = 3 // D
	// Categoria não foi mapeada propositalmente, pois não é necessário para a lógica de conciliação, mas pode ser adicionado se necessário.
	ColumnDono       = 5 // F
	ColumnBanco      = 6 // G
	ColumnConta      = 7 // H
	ColumnRecorrente = 8 // I
	ColumnIdParcela  = 9 // J
)

// Transaction represents a row in the spreadsheet (ES or DIF)
type Transaction struct {
	RowIndex   int     `json:"rowIndex"` // 0-based index in the sheet
	Dono       string  `json:"dono"`
	Banco      string  `json:"banco"`
	Conta      string  `json:"conta"`
	Descricao  string  `json:"descricao"`
	Recorrente bool    `json:"recorrente"`
	Data       string  `json:"data"`
	Valor      float64 `json:"valor"`
	IdParcela  string  `json:"idParcela"`
	Sheet      string  `json:"sheet"` // "ES" or "DIF"
}

// ConciliationCandidate represents a potential match
type ConciliationCandidate struct {
	Reference  Transaction   `json:"reference"`  // From DIF
	Candidates []Transaction `json:"candidates"` // From ES
}

// PendingConciliationSummary is a lightweight view for the list
type PendingConciliationSummary struct {
	DifRowIndex    int     `json:"difRowIndex"`
	IdParcela      string  `json:"idParcela"`
	Dono           string  `json:"dono"`
	Banco          string  `json:"banco"`
	Conta          string  `json:"conta"`
	Descricao      string  `json:"descricao"`
	Data           string  `json:"data"`
	Valor          float64 `json:"valor"`
	CandidateCount int     `json:"candidateCount"`
}

// AcceptRequest defines the body for accepting a conciliation
type AcceptRequest struct {
	EsRowIndices []int `json:"esRowIndices"`
}
