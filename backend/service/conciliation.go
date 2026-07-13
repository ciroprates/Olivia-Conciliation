package service

import (
	"errors"
	"math"
	"strings"

	"olivia-conciliation/backend/config"
	"olivia-conciliation/backend/models"
)

// ErrTransactionNotInHOM sinaliza que nenhuma linha da HOM tem o IdParcela pedido.
// Ocorre só se um Processamento de Transações reescrever a HOM entre listar e salvar
// e o Pluggy não trouxer mais aquela transação. Os handlers mapeiam para HTTP 404.
var ErrTransactionNotInHOM = errors.New("transaction not found in HOM")

// ErrEmptyIdParcela sinaliza um pedido de edição sem IdParcela. Mapeado para HTTP 400.
var ErrEmptyIdParcela = errors.New("idParcela is required")

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

// findHOMRowByIdParcela localiza na HOM a linha cujo IdParcela é igual ao pedido.
// Como o IdParcela é único (ver CONTEXT.md), retorna no máximo uma linha.
// Endereçar por identidade — e não pelo índice da DIF — evita o descasamento do #21:
// a DIF é gerada por FILTER sobre a HOM, então os índices raramente coincidem.
func (l *Logic) findHOMRowByIdParcela(idParcela string) (int, error) {
	target := strings.TrimSpace(idParcela)
	if target == "" {
		return 0, ErrEmptyIdParcela
	}

	homRows, err := l.repo.FetchRows(l.cfg.SheetHOM)
	if err != nil {
		return 0, err
	}

	for i := 1; i < len(homRows); i++ {
		row := homRows[i]
		if l.parser.IsEmpty(row) {
			continue
		}
		hom := l.parser.ParseTransaction(i, row, "HOM")
		if strings.TrimSpace(hom.IdParcela) == target {
			return i, nil
		}
	}
	return 0, ErrTransactionNotInHOM
}

func (l *Logic) UpdateDifCategory(idParcela, categoria string) error {
	rowIdx, err := l.findHOMRowByIdParcela(idParcela)
	if err != nil {
		return err
	}
	return l.repo.WriteCell(l.cfg.SheetHOM, rowIdx, models.ColumnCategoria, categoria)
}

func (l *Logic) UpdateDifDate(idParcela, data string) error {
	rowIdx, err := l.findHOMRowByIdParcela(idParcela)
	if err != nil {
		return err
	}
	return l.repo.WriteCell(l.cfg.SheetHOM, rowIdx, models.ColumnData, data)
}
