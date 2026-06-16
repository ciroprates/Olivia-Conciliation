package config

import "testing"

func TestFromEnv_CookieSecure_DefaultTrue(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "")
	cfg := FromEnv()
	if !cfg.CookieSecure {
		t.Error("CookieSecure should be true when COOKIE_SECURE is unset")
	}
}

func TestFromEnv_CookieSecure_FalseWhenExplicit(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "false")
	cfg := FromEnv()
	if cfg.CookieSecure {
		t.Error("CookieSecure should be false when COOKIE_SECURE=false")
	}
}

func TestFromEnv_CookieSecure_TrueForOtherValues(t *testing.T) {
	for _, val := range []string{"true", "1", "yes"} {
		t.Setenv("COOKIE_SECURE", val)
		if !FromEnv().CookieSecure {
			t.Errorf("CookieSecure should be true for COOKIE_SECURE=%q", val)
		}
	}
}

func TestFromEnv_CookieSecure_FalseWithWhitespace(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "  FALSE  ")
	if FromEnv().CookieSecure {
		t.Error("CookieSecure should be false for COOKIE_SECURE='  FALSE  ' (trimmed+lowercased)")
	}
}

func TestFromEnv_ReadsSheetNames(t *testing.T) {
	t.Setenv("SHEET_ES", "Entradas e Saídas")
	t.Setenv("SHEET_DIF", "Diferença")
	t.Setenv("SHEET_REJ", "Rejeitados")
	t.Setenv("SHEET_HOM", "Homologação")
	cfg := FromEnv()
	if cfg.SheetES != "Entradas e Saídas" {
		t.Errorf("SheetES=%q", cfg.SheetES)
	}
	if cfg.SheetDIF != "Diferença" {
		t.Errorf("SheetDIF=%q", cfg.SheetDIF)
	}
	if cfg.SheetREJ != "Rejeitados" {
		t.Errorf("SheetREJ=%q", cfg.SheetREJ)
	}
	if cfg.SheetHOM != "Homologação" {
		t.Errorf("SheetHOM=%q", cfg.SheetHOM)
	}
}
