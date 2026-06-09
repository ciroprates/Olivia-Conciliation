package handlers

import (
	"testing"
)

func TestExtractPathID(t *testing.T) {
	cases := []struct {
		path    string
		depth   int
		wantID  int
		wantErr bool
	}{
		{"/api/conciliations/42", 1, 42, false},
		{"/api/conciliations/42/accept", 2, 42, false},
		{"/api/dif/non-recurring/7/move-to-es", 2, 7, false},
		{"/short", 2, 0, true},
		{"/api/conciliations/abc", 1, 0, true},
	}

	for _, c := range cases {
		got, err := extractPathID(c.path, c.depth)
		if c.wantErr {
			if err == nil {
				t.Errorf("extractPathID(%q, %d): expected error, got %d", c.path, c.depth, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("extractPathID(%q, %d): unexpected error: %v", c.path, c.depth, err)
			continue
		}
		if got != c.wantID {
			t.Errorf("extractPathID(%q, %d) = %d, want %d", c.path, c.depth, got, c.wantID)
		}
	}
}
