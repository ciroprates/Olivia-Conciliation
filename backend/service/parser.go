package service

import (
	"fmt"
	"strconv"
	"strings"

	"olivia-conciliation/backend/models"
)

// Parser converts raw spreadsheet rows into domain types.
type Parser struct{}

func (p Parser) ParseFloat(v interface{}) float64 {
	if v == nil {
		return 0
	}
	s := strings.ReplaceAll(fmt.Sprintf("%v", v), "R$", "")
	s = strings.TrimSpace(s)
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// pt-BR format: 1.000,00 → remove thousands dot, replace decimal comma
	s2 := strings.ReplaceAll(s, ".", "")
	s2 = strings.ReplaceAll(s2, ",", ".")
	if f, err := strconv.ParseFloat(s2, 64); err == nil {
		return f
	}
	return 0
}

func (p Parser) ParseBool(v interface{}) bool {
	if v == nil {
		return false
	}
	s := strings.ToLower(fmt.Sprintf("%v", v))
	return s == "sim" || s == "yes" || s == "true"
}

func (p Parser) IsEmpty(row []interface{}) bool {
	if len(row) == 0 {
		return true
	}
	for _, cell := range row {
		if strings.TrimSpace(fmt.Sprintf("%v", cell)) != "" {
			return false
		}
	}
	return true
}

// IsPending reports whether a transaction in ES is still awaiting conciliation.
func (p Parser) IsPending(t models.Transaction) bool {
	idParcela := strings.ToLower(strings.TrimSpace(t.IdParcela))
	if strings.HasPrefix(idParcela, "synthetic") {
		return true
	}
	return t.Recorrente && idParcela == ""
}

func (p Parser) ParseTransaction(idx int, row []interface{}, sheetName string) models.Transaction {
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
		t.Recorrente = p.ParseBool(row[models.ColumnRecorrente])
	}
	if len(row) > models.ColumnData {
		t.Data = fmt.Sprintf("%v", row[models.ColumnData])
	}
	if len(row) > models.ColumnValor {
		t.Valor = p.ParseFloat(row[models.ColumnValor])
	}
	if len(row) > models.ColumnCategoria {
		t.Categoria = fmt.Sprintf("%v", row[models.ColumnCategoria])
	}
	if len(row) > models.ColumnIdParcela {
		t.IdParcela = fmt.Sprintf("%v", row[models.ColumnIdParcela])
	}
	return t
}
