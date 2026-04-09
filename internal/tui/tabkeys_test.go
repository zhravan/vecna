package tui

import "testing"

func TestTabIndexFromPlainDigitKey(t *testing.T) {
	for _, tt := range []struct {
		s    string
		want int
		ok   bool
	}{
		{s: "1", want: 0, ok: true},
		{s: "9", want: 8, ok: true},
		{s: "0", want: 0, ok: false},
		{s: "11", want: 0, ok: false},
		{s: "alt+1", want: 0, ok: false},
	} {
		got, ok := tabIndexFromPlainDigitKey(tt.s)
		if ok != tt.ok || got != tt.want {
			t.Errorf("tabIndexFromPlainDigitKey(%q) = (%d, %v), want (%d, %v)", tt.s, got, ok, tt.want, tt.ok)
		}
	}
}

func TestTabIndexFromAltDigitKey(t *testing.T) {
	tests := []struct {
		s    string
		want int
		ok   bool
	}{
		{s: "alt+1", want: 0, ok: true},
		{s: "alt+9", want: 8, ok: true},
		{s: "1", want: 0, ok: false},
		{s: "alt+0", want: 0, ok: false},
		{s: "alt+a", want: 0, ok: false},
	}
	for _, tt := range tests {
		got, ok := tabIndexFromAltDigitKey(tt.s)
		if ok != tt.ok || got != tt.want {
			t.Errorf("tabIndexFromAltDigitKey(%q) = (%d, %v), want (%d, %v)", tt.s, got, ok, tt.want, tt.ok)
		}
	}
}

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
