package tui

import "testing"

func TestParseKittySuperDigitTab(t *testing.T) {
	tests := []struct {
		seq  string
		want int
		ok   bool
	}{
		{seq: "\x1b[49;8u", want: 0, ok: true},
		{seq: "\x1b[57;8u", want: 8, ok: true},
		{seq: "\x1b[49;8:1u", want: 0, ok: true},
		{seq: "\x1b[49;8:3u", want: 0, ok: false},
		{seq: "\x1b[49;0u", want: 0, ok: false},
		{seq: "\x1b[65;8u", want: 0, ok: false},
		{seq: "nope", want: 0, ok: false},
	}
	for _, tt := range tests {
		got, ok := parseKittySuperDigitTab([]byte(tt.seq))
		if ok != tt.ok || got != tt.want {
			t.Errorf("parseKittySuperDigitTab(%q) = (%d, %v), want (%d, %v)", tt.seq, got, ok, tt.want, tt.ok)
		}
	}
}
