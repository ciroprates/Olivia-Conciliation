package service

import (
	"errors"
	"math"
	"os"

	"olivia-conciliation/backend/models"
	"olivia-conciliation/backend/sheets"
)

type Logic struct {
	client *sheets.Client
	parser Parser
}

func NewLogic(client *sheets.Client) *Logic {
	return &Logic{client: client}
}

func isMatch(dif, es models.Transaction) bool {
	if dif.Dono != es.Dono || dif.Banco != es.Banco || dif.Conta != es.Conta {
		return false
	}
	return math.Abs(dif.Valor-es.Valor) < 5.00
}

func (l *Logic) GetConciliations() ([]models.PendingConciliationSummary, error) {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}
	esRows, err := l.client.FetchRows(os.Getenv("SHEET_ES"))
	if err != nil {
		return nil, err
	}

	var candidates []models.Transaction
	for i := 1; i < len(esRows); i++ {
		t := l.parser.ParseTransaction(i, esRows[i], "ES")
		if l.parser.IsPending(t) {
			candidates = append(candidates, t)
		}
	}

	var results []models.PendingConciliationSummary
	for i := 1; i < len(difRows); i++ {
		dif := l.parser.ParseTransaction(i, difRows[i], "DIF")
		if dif.Dono == "" && dif.Valor == 0 {
			continue
		}
		if !dif.Recorrente {
			continue
		}

		count := 0
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

func (l *Logic) GetConciliationDetails(difIndex int) (*models.ConciliationCandidate, error) {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}
	if difIndex >= len(difRows) {
		return nil, errors.New("DIF index out of bounds")
	}

	dif := l.parser.ParseTransaction(difIndex, difRows[difIndex], "DIF")
	if !dif.Recorrente {
		return nil, errors.New("DIF transaction is not recurring")
	}

	esRows, err := l.client.FetchRows(os.Getenv("SHEET_ES"))
	if err != nil {
		return nil, err
	}

	var matchCandidates []models.Transaction
	for i := 1; i < len(esRows); i++ {
		t := l.parser.ParseTransaction(i, esRows[i], "ES")
		if l.parser.IsPending(t) && isMatch(dif, t) {
			matchCandidates = append(matchCandidates, t)
		}
	}

	return &models.ConciliationCandidate{
		Reference:  dif,
		Candidates: matchCandidates,
	}, nil
}

func (l *Logic) Accept(difIndex int, esIndices []int) error {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	dif := l.parser.ParseTransaction(difIndex, difRows[difIndex], "DIF")
	if dif.IdParcela == "" {
		return errors.New("DIF transaction has no ID")
	}

	for _, esIdx := range esIndices {
		if err := l.client.WriteCell(os.Getenv("SHEET_ES"), esIdx, models.ColumnIdParcela, dif.IdParcela); err != nil {
			return err
		}
	}
	return nil
}

func (l *Logic) Reject(difIndex int) error {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	rowContent := difRows[difIndex]
	if err := l.client.AppendRow(os.Getenv("SHEET_REJ"), rowContent); err != nil {
		return err
	}
	return l.client.ClearRow(os.Getenv("SHEET_DIF"), difIndex)
}

func (l *Logic) ListNonRecurringDIF() ([]models.NonRecurringDifSummary, error) {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}

	results := make([]models.NonRecurringDifSummary, 0)
	for i := 1; i < len(difRows); i++ {
		row := difRows[i]
		if l.parser.IsEmpty(row) {
			continue
		}

		dif := l.parser.ParseTransaction(i, row, "DIF")
		if dif.Recorrente {
			continue
		}

		results = append(results, models.NonRecurringDifSummary{
			DifRowIndex: dif.RowIndex,
			Dono:        dif.Dono,
			Banco:       dif.Banco,
			Conta:       dif.Conta,
			Descricao:   dif.Descricao,
			Data:        dif.Data,
			Valor:       dif.Valor,
			Categoria:   dif.Categoria,
			IdParcela:   dif.IdParcela,
		})
	}

	return results, nil
}

func (l *Logic) MoveNonRecurringDifToES(difIndex int) error {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	rowContent := difRows[difIndex]
	if l.parser.IsEmpty(rowContent) {
		return errors.New("DIF row is empty")
	}

	dif := l.parser.ParseTransaction(difIndex, rowContent, "DIF")
	if dif.Recorrente {
		return errors.New("DIF transaction is recurring")
	}

	if err := l.client.AppendRow(os.Getenv("SHEET_ES"), rowContent); err != nil {
		return err
	}
	return l.client.ClearRow(os.Getenv("SHEET_DIF"), difIndex)
}

func (l *Logic) MoveNonRecurringDifToREJ(difIndex int) error {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	rowContent := difRows[difIndex]
	if l.parser.IsEmpty(rowContent) {
		return errors.New("DIF row is empty")
	}

	dif := l.parser.ParseTransaction(difIndex, rowContent, "DIF")
	if dif.Recorrente {
		return errors.New("DIF transaction is recurring")
	}

	if err := l.client.AppendRow(os.Getenv("SHEET_REJ"), rowContent); err != nil {
		return err
	}
	return l.client.ClearRow(os.Getenv("SHEET_DIF"), difIndex)
}

func (l *Logic) MoveAllNonRecurringDifToES() (*models.NonRecurringBulkActionResult, error) {
	difRows, err := l.client.FetchRows(os.Getenv("SHEET_DIF"))
	if err != nil {
		return nil, err
	}

	moved := 0
	for i := 1; i < len(difRows); i++ {
		rowContent := difRows[i]
		if l.parser.IsEmpty(rowContent) {
			continue
		}

		dif := l.parser.ParseTransaction(i, rowContent, "DIF")
		if dif.Recorrente {
			continue
		}

		if err := l.client.AppendRow(os.Getenv("SHEET_ES"), rowContent); err != nil {
			return nil, err
		}
		if err := l.client.ClearRow(os.Getenv("SHEET_DIF"), i); err != nil {
			return nil, err
		}
		moved++
	}

	return &models.NonRecurringBulkActionResult{MovedToES: moved}, nil
}

func (l *Logic) UpdateDifCategory(difIndex int, categoria string) error {
	homRows, err := l.client.FetchRows(os.Getenv("SHEET_HOM"))
	if err != nil {
		return err
	}
	if difIndex >= len(homRows) {
		return errors.New("index out of bounds")
	}

	if l.parser.IsEmpty(homRows[difIndex]) {
		return errors.New("row is empty")
	}

	return l.client.WriteCell(os.Getenv("SHEET_HOM"), difIndex, models.ColumnCategoria, categoria)
}

func (l *Logic) UpdateDifDate(difIndex int, data string) error {
	if difIndex < 0 {
		return errors.New("invalid index")
	}
	return l.client.WriteCell(os.Getenv("SHEET_HOM"), difIndex, models.ColumnData, data)
}
