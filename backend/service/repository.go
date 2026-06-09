package service

// SheetRepository is the persistence boundary between service logic and Google Sheets.
// sheets.Client satisfies this interface in production; in-memory adapters are used in tests.
type SheetRepository interface {
	FetchRows(sheet string) ([][]interface{}, error)
	WriteCell(sheet string, rowIdx, colIdx int, value string) error
	AppendRow(sheet string, values []interface{}) error
	ClearRow(sheet string, rowIdx int) error
}
