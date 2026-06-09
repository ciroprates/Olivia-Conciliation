package service

import (
	"errors"
	"math"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/models"
)

type Logic struct {
	repo   SheetRepository
	cfg    config.Config
	parser Parser
}

func NewLogic(repo SheetRepository, cfg config.Config) *Logic {
	return &Logic{repo: repo, cfg: cfg}
}

func isMatch(dif, es models.Transaction) bool {
	if dif.Dono != es.Dono || dif.Banco != es.Banco || dif.Conta != es.Conta {
		return false
	}
	return math.Abs(dif.Valor-es.Valor) < 5.00
}

func (l *Logic) GetConciliations() ([]models.PendingConciliationSummary, error) {
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
	if err != nil {
		return nil, err
	}
	esRows, err := l.repo.FetchRows(l.cfg.SheetES)
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
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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

	esRows, err := l.repo.FetchRows(l.cfg.SheetES)
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
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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
		if err := l.repo.WriteCell(l.cfg.SheetES, esIdx, models.ColumnIdParcela, dif.IdParcela); err != nil {
			return err
		}
	}
	return nil
}

func (l *Logic) Reject(difIndex int) error {
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
	if err != nil {
		return err
	}
	if difIndex >= len(difRows) {
		return errors.New("index out of bounds")
	}

	rowContent := difRows[difIndex]
	if err := l.repo.AppendRow(l.cfg.SheetREJ, rowContent); err != nil {
		return err
	}
	return l.repo.ClearRow(l.cfg.SheetDIF, difIndex)
}

func (l *Logic) ListNonRecurringDIF() ([]models.NonRecurringDifSummary, error) {
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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

	if err := l.repo.AppendRow(l.cfg.SheetES, rowContent); err != nil {
		return err
	}
	return l.repo.ClearRow(l.cfg.SheetDIF, difIndex)
}

func (l *Logic) MoveNonRecurringDifToREJ(difIndex int) error {
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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

	if err := l.repo.AppendRow(l.cfg.SheetREJ, rowContent); err != nil {
		return err
	}
	return l.repo.ClearRow(l.cfg.SheetDIF, difIndex)
}

func (l *Logic) MoveAllNonRecurringDifToES() (*models.NonRecurringBulkActionResult, error) {
	difRows, err := l.repo.FetchRows(l.cfg.SheetDIF)
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

		if err := l.repo.AppendRow(l.cfg.SheetES, rowContent); err != nil {
			return nil, err
		}
		if err := l.repo.ClearRow(l.cfg.SheetDIF, i); err != nil {
			return nil, err
		}
		moved++
	}

	return &models.NonRecurringBulkActionResult{MovedToES: moved}, nil
}

func (l *Logic) UpdateDifCategory(difIndex int, categoria string) error {
	homRows, err := l.repo.FetchRows(l.cfg.SheetHOM)
	if err != nil {
		return err
	}
	if difIndex >= len(homRows) {
		return errors.New("index out of bounds")
	}

	if l.parser.IsEmpty(homRows[difIndex]) {
		return errors.New("row is empty")
	}

	return l.repo.WriteCell(l.cfg.SheetHOM, difIndex, models.ColumnCategoria, categoria)
}

func (l *Logic) UpdateDifDate(difIndex int, data string) error {
	if difIndex < 0 {
		return errors.New("invalid index")
	}
	return l.repo.WriteCell(l.cfg.SheetHOM, difIndex, models.ColumnData, data)
}
