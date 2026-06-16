package config

import (
	"os"
	"strings"
)

// Config holds all configuration read from environment variables at startup.
type Config struct {
	SpreadsheetID string
	SheetES       string
	SheetDIF      string
	SheetREJ      string
	SheetHOM      string
	AdminUser     string
	AdminPass     string
	JWTSecret     string
	AppOrigin     string
	CookieDomain  string
	CookieSecure  bool
}

func FromEnv() Config {
	return Config{
		SpreadsheetID: os.Getenv("SHEET_SPREADSHEET_ID"),
		SheetES:       os.Getenv("SHEET_ES"),
		SheetDIF:      os.Getenv("SHEET_DIF"),
		SheetREJ:      os.Getenv("SHEET_REJ"),
		SheetHOM:      os.Getenv("SHEET_HOM"),
		AdminUser:     os.Getenv("ADMIN_USER"),
		AdminPass:     os.Getenv("ADMIN_PASS"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		AppOrigin:     os.Getenv("APP_ORIGIN"),
		CookieDomain:  os.Getenv("COOKIE_DOMAIN"),
		CookieSecure:  strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SECURE"))) != "false",
	}
}
