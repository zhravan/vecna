package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTabSwitchDeltaFromKey(t *testing.T) {
	tests := map[string]int{"ctrl+tab": 1, "ctrl+shift+tab": -1, "ctrl+right": 1, "ctrl+left": -1, "alt+right": 1, "alt+left": -1, "enter": 0}
	for key, want := range tests {
		got, ok := tabSwitchDeltaFromKey(key)
		if want == 0 {
			if ok {
				t.Fatalf("%q: expected no match", key)
			}
			continue
		}
		if !ok || got != want {
			t.Fatalf("%q: got (%d,%v), want (%d,true)", key, got, ok, want)
		}
	}
}

func TestNormalizeDroppedPaths(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "hello world.txt")
	second := filepath.Join(dir, "second.txt")
	if err := os.WriteFile(first, []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("two"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, in string
		want     int
	}{
		{"quoted single", "'" + first + "'", 1},
		{"quoted double", "\"" + first + "\"", 1},
		{"escaped space", strings.ReplaceAll(first, " ", "\\ "), 1},
		{"multiple", "'" + first + "' '" + second + "'", 2},
		{"missing", filepath.Join(dir, "missing.txt"), 0},
		{"plain text", "hello world", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDroppedPaths(tt.in)
			if len(got) != tt.want {
				t.Fatalf("got %v, want %d paths", got, tt.want)
			}
		})
	}
}


func TestRenderTabBarKeepsVisibleLabels(t *testing.T) {
	m := Model{
		width: 80,
		tabs: []tab{
			{Id: 0, Kind: tabKindHome, Title: "Hosts"},
			{Id: 1, Kind: tabKindSSH, Title: "production"},
		},
		currentTabIndex: 1,
	}
	got := m.renderTabBar()
	if !strings.Contains(got, "Hosts") || !strings.Contains(got, "production") {
		t.Fatalf("tab bar lost visible labels: %q", got)
	}
	if h := lipgloss.Height(got); h != 1 {
		t.Fatalf("tab bar height = %d, want 1", h)
	}
	if w := lipgloss.Width(got); w != 80 {
		t.Fatalf("tab bar width = %d, want 80", w)
	}
}
