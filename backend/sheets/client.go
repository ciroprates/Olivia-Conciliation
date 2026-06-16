package sheets

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type Client struct {
	srv           *sheets.Service
	spreadsheetID string
	tableIDs      map[string]string
}

// NewClient creates a Sheets client and caches the native table ID for each sheet
// listed in tableSheets. Fails if any sheet has zero or more than one native table.
func NewClient(ctx context.Context, spreadsheetID string, tableSheets ...string) (*Client, error) {
	credsPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if credsPath == "" {
		credsPath = "credentials.json"
	}

	b, err := os.ReadFile(credsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.JWTConfigFromJSON(b, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	c := &Client{
		srv:           nil,
		spreadsheetID: spreadsheetID,
		tableIDs:      make(map[string]string),
	}

	svc, err := sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Sheets client: %v", err)
	}
	c.srv = svc

	if len(tableSheets) > 0 {
		if err := c.initTableIDs(tableSheets); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Client) initTableIDs(sheetNames []string) error {
	sp, err := c.srv.Spreadsheets.Get(c.spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("unable to get spreadsheet metadata: %v", err)
	}

	byName := make(map[string]*sheets.Sheet, len(sp.Sheets))
	for _, s := range sp.Sheets {
		byName[s.Properties.Title] = s
	}

	for _, name := range sheetNames {
		s, ok := byName[name]
		if !ok {
			return fmt.Errorf("sheet %q not found in spreadsheet", name)
		}
		if len(s.Tables) != 1 {
			return fmt.Errorf("sheet %q must have exactly one native table, found %d", name, len(s.Tables))
		}
		c.tableIDs[name] = s.Tables[0].TableId
	}

	return nil
}

func (c *Client) FetchRows(sheetName string) ([][]interface{}, error) {
	resp, err := c.srv.Spreadsheets.Values.Get(c.spreadsheetID, sheetName).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve data from sheet: %v", err)
	}
	return resp.Values, nil
}

func (c *Client) WriteCell(sheetName string, rowIndex int, colIndex int, value string) error {
	colLetter := ""
	tempIdx := colIndex
	for {
		colLetter = string(rune('A'+(tempIdx%26))) + colLetter
		tempIdx = (tempIdx / 26) - 1
		if tempIdx < 0 {
			break
		}
	}

	rangeStr := fmt.Sprintf("%s!%s%d", sheetName, colLetter, rowIndex+1)
	val := &sheets.ValueRange{
		Values: [][]interface{}{{value}},
	}

	_, err := c.srv.Spreadsheets.Values.Update(c.spreadsheetID, rangeStr, val).ValueInputOption("RAW").Do()
	if err != nil {
		return fmt.Errorf("unable to update data: %v", err)
	}
	return nil
}

func (c *Client) AppendRow(sheetName string, values []interface{}) error {
	tableID, ok := c.tableIDs[sheetName]
	if !ok {
		return fmt.Errorf("no native table cached for sheet %q", sheetName)
	}

	cellData := make([]*sheets.CellData, len(values))
	for i, v := range values {
		str := ""
		if v != nil {
			str = fmt.Sprintf("%v", v)
		}
		ev := &sheets.ExtendedValue{StringValue: &str}
		if str == "" {
			ev.ForceSendFields = []string{"StringValue"}
		}
		cellData[i] = &sheets.CellData{UserEnteredValue: ev}
	}

	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				AppendCells: &sheets.AppendCellsRequest{
					TableId: tableID,
					Rows:    []*sheets.RowData{{Values: cellData}},
					Fields:  "userEnteredValue",
				},
			},
		},
	}

	_, err := c.srv.Spreadsheets.BatchUpdate(c.spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("unable to append row to sheet %q: %v", sheetName, err)
	}
	return nil
}

func (c *Client) ClearRow(sheetName string, rowIndex int) error {
	rangeStr := fmt.Sprintf("%s!A%d:ZZ%d", sheetName, rowIndex+1, rowIndex+1)
	_, err := c.srv.Spreadsheets.Values.Clear(c.spreadsheetID, rangeStr, &sheets.ClearValuesRequest{}).Do()
	return err
}
