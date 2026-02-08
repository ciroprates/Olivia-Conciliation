package service

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"olivia-conciliation/backend/models"
	"olivia-conciliation/backend/sheets"
)

type Logic struct {
	client *sheets.Client
}

func NewLogic(client *sheets.Client) *Logic {
	return &Logic{client: client}
}

// Helper: Parse float from string, handling "R$ 1.234,56" or "1234.56"
func parseFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	s := fmt.Sprintf("%v", v)
	// Simple cleanup: remove R$, space. Replace comma with dot if needed?
	// Assuming standard format. If comma is decimal separator (pt-BR), replace . with nothing and , with .
	// But Sheets API ValueRenderOption="UNFORMATTED_VALUE" gives float/int.
	// We are using default which might be FORMATTED_VALUE.
	// Let's blindly try to parse, or assume raw numbers if unformatted.
	// The previous fetching code didn't specify ValueRenderOption. Default is FORMATTED.
	// Let's change client to UNFORMATTED_VALUE would be better, but let's just clean strings.

	s = strings.ReplaceAll(s, "R$", "")
	s = strings.TrimSpace(s)
	// Try simple parse
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f
	}
	// Try pt-BR: 1.000,00 -> remove dot, replace comma with dot
	s2 := strings.ReplaceAll(s, ".", "")
	s2 = strings.ReplaceAll(s2, ",", ".")
	f, err = strconv.ParseFloat(s2, 64)
	if err == nil {
		return f
	}
	return 0
}

func parseBool(v interface{}) bool {
	if v == nil {
		return false
	}
	s := strings.ToLower(fmt.Sprintf("%v", v))
	return s == "sim" || s == "yes" || s == "true"
}

func rowToTransaction(idx int, row []interface{}, sheetName string) models.Transaction {
	t := models.Transaction{RowIndex: idx, Sheet: sheetName}
	if len(row) > models.ColumnDono {
		t.Dono = fmt.Sprintf("%v", row[models.ColumnDono])
	}
	if len(row) > models.ColumnBanco {
		t.Banco = fmt.Sprintf("%v", row[models.ColumnBanco])
	}
	if len(row) > models.ColumnConta {
		t.Conta = fmt.Sprintf("%v", row[models.ColumnConta])
	}
	if len(row) > models.ColumnDescricao {
		t.Descricao = fmt.Sprintf("%v", row[models.ColumnDescricao])
	}
	if len(row) > models.ColumnRecorrente {
		t.Recorrente = parseBool(row[models.ColumnRecorrente])
	}
	if len(row) > models.ColumnData {
		t.Data = fmt.Sprintf("%v", row[models.ColumnData])
	}
	if len(row) > models.ColumnValor {
		t.Valor = parseFloat(row[models.ColumnValor])
	}
	if len(row) > models.ColumnIdParcela {
		t.IdParcela = fmt.Sprintf("%v", row[models.ColumnIdParcela])
	}
	return t
}

func (l *Logic) GetConciliations() ([]models.PendingConciliationSummary, error) {
	// 1. Fetch DIF
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}

	// 2. Fetch ES
	esRows, err := l.client.FetchRows(os.Getenv("SHEET_ES"))
	if err != nil {
		return nil, err
	}

	var results []models.PendingConciliationSummary

	// Skip header row if present? Assuming Row 1 is header, so data starts at index 1 (Line 2)
	// Prompt says "Índice (A=1)". Usually row 1 is headers.
	// Let's start loop from 1.

	// Parse ES candidates first optimization
	var candidates []models.Transaction
	for i := 1; i < len(esRows); i++ {
		t := rowToTransaction(i, esRows[i], "ES")
		// Filter: Recorrente? = "Sim" e Id da parcela vazio
		if t.Recorrente && t.IdParcela == "" {
			candidates = append(candidates, t)
		}
	}

	// Iterate DIF
	for i := 1; i < len(difRows); i++ {
		dif := rowToTransaction(i, difRows[i], "DIF")
		// DIF must have ID? "Diferença (DIF): transações futuras com ID".
		// We assume valid DIF row has content.
		if dif.Dono == "" && dif.Valor == 0 {
			continue
		} // Skip empty

		count := 0
		// Match logic
		for _, es := range candidates {
			if isMatch(dif, es) {
				count++
			}
		}

		results = append(results, models.PendingConciliationSummary{
			DifRowIndex:    dif.RowIndex,
			IdParcela:      dif.IdParcela,
			Dono:           dif.Dono,
			Banco:          dif.Banco,
			Conta:          dif.Conta,
			Descricao:      dif.Descricao,
			Data:           dif.Data,
			Valor:          dif.Valor,
			CandidateCount: count,
		})
	}
	return results, nil
}

func isMatch(dif, es models.Transaction) bool {
	// Exact match
	if dif.Dono != es.Dono {
		return false
	}
	if dif.Banco != es.Banco {
		return false
	}
	if dif.Conta != es.Conta {
		return false
	}

	// Soft match: abs(Diff - ES) < 5.00
	diffVal := math.Abs(dif.Valor - es.Valor)
	if diffVal >= 5.00 {
		return false
	}

	return true
}

func (l *Logic) GetConciliationDetails(difIndex int) (*models.ConciliationCandidate, error) {
	// Fetch all again? Optimization: could pass context, but simpler to fetch.
	// For single detail, we need the specific DIF row and ALL ES to search.

	// 1. Fetch DIF row
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}
	if difIndex >= len(difRows) {
		return nil, errors.New("DIF index out of bounds")
	}

	dif := rowToTransaction(difIndex, difRows[difIndex], "DIF")

	// 2. Fetch ES
	esRows, err := l.client.FetchRows(os.Getenv("SHEET_ES"))
	if err != nil {
		return nil, err
	}

	var matchCandidates []models.Transaction
	for i := 1; i < len(esRows); i++ {
		t := rowToTransaction(i, esRows[i], "ES")
		if t.Recorrente && t.IdParcela == "" {
			if isMatch(dif, t) {
				matchCandidates = append(matchCandidates, t)
			}
		}
	}

	return &models.ConciliationCandidate{
		Reference:  dif,
		Candidates: matchCandidates,
	}, nil
}

func (l *Logic) Accept(difIndex int, esIndices []int) error {
	// 1. Get DIF to find ID
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	dif := rowToTransaction(difIndex, difRows[difIndex], "DIF")
	if dif.IdParcela == "" {
		return errors.New("DIF transaction has no ID")
	}

	// 2. Update ES rows
	for _, esIdx := range esIndices {
		// Write ID to ColumnIdParcela (Col H = index 7)
		err := l.client.WriteCell(os.Getenv("SHEET_ES"), esIdx, models.ColumnIdParcela, dif.IdParcela)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Logic) Reject(difIndex int) error {
	// 1. Get DIF Row to move
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	// row content slice
	rowContent := difRows[difIndex]

	// 2. Append to REJ
	err = l.client.AppendRow(os.Getenv("SHEET_REJ"), rowContent)
	if err != nil {
		return err
	}

	// 3. Clear/Delete DIF Row
	return l.client.ClearRow(os.Getenv("SHEET_DIF"), difIndex)
}
