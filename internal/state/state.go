package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// maxRecent caps how many host names we keep in recency order.
const maxRecent = 40

// State is persisted separately from config.yaml so recents can update often
// without rewriting the main host definitions.
type State struct {
	RecentHostNames []string `json:"recent_host_names"`
}

// Path returns ~/.config/vecna/state.json
func Path() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "state.json")
	}
	return filepath.Join(home, ".config", "vecna", "state.json")
}

// Load reads state from disk; missing or invalid file yields empty state.
func Load() State {
	p := Path()
	b, err := os.ReadFile(p)
	if err != nil {
		return State{}
	}
	var s State
	if json.Unmarshal(b, &s) != nil {
		return State{}
	}
	return s
}

// Save writes state with restrictive permissions.
func Save(s State) error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0600)
}

// RecordRecent returns a new state with hostName moved to the front of recents.
func RecordRecent(s State, hostName string) State {
	if hostName == "" {
		return s
	}
	filtered := make([]string, 0, len(s.RecentHostNames)+1)
	for _, n := range s.RecentHostNames {
		if n != hostName {
			filtered = append(filtered, n)
		}
	}
	out := State{RecentHostNames: append([]string{hostName}, filtered...)}
	if len(out.RecentHostNames) > maxRecent {
		out.RecentHostNames = out.RecentHostNames[:maxRecent]
	}
	return out
}

// RemoveHost returns a new state without hostName in recents.
func RemoveHost(s State, hostName string) State {
	if hostName == "" {
		return s
	}
	filtered := make([]string, 0, len(s.RecentHostNames))
	for _, n := range s.RecentHostNames {
		if n != hostName {
			filtered = append(filtered, n)
		}
	}
	return State{RecentHostNames: filtered}
}

// RenameHost replaces oldName with newName in the recent list order.
func RenameHost(s State, oldName, newName string) State {
	if oldName == "" || newName == "" || oldName == newName {
		return s
	}
	next := make([]string, len(s.RecentHostNames))
	for i, n := range s.RecentHostNames {
		if n == oldName {
			next[i] = newName
		} else {
			next[i] = n
		}
	}
	// Dedupe if newName already appeared elsewhere
	seen := make(map[string]bool)
	var deduped []string
	for _, n := range next {
		if n == "" {
			continue
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		deduped = append(deduped, n)
	}
	return State{RecentHostNames: deduped}
}
