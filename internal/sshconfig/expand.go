package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandIncludes recursively expands OpenSSH Include directives. Globs are
// resolved relative to the including file and duplicate files are ignored.
func ExpandIncludes(path string) ([]byte, error) {
	seen := map[string]bool{}
	return expandFile(path, seen)
}

func expandFile(path string, seen map[string]bool) ([]byte, error) {
	path = expandHome(path)
	abs, err := filepath.Abs(path); if err != nil { return nil, err }
	if seen[abs] { return nil, nil }; seen[abs] = true
	b, err := os.ReadFile(abs); if err != nil { return nil, err }
	base := filepath.Dir(abs); var out strings.Builder
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if len(trim) == 0 || strings.HasPrefix(trim, "#") { out.WriteString(line); out.WriteByte('\n'); continue }
		fields := strings.Fields(trim)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Include") {
			for _, pattern := range fields[1:] {
				if strings.HasPrefix(pattern, "~") { pattern = expandHome(pattern) } else if !filepath.IsAbs(pattern) { pattern = filepath.Join(base, pattern) }
				matches, _ := filepath.Glob(pattern)
				if len(matches) == 0 { continue }
				for _, match := range matches { child, e := expandFile(match, seen); if e != nil { return nil, fmt.Errorf("include %s: %w", match, e) }; out.Write(child) }
			}
			continue
		}
		out.WriteString(line); out.WriteByte('\n')
	}
	return []byte(out.String()), nil
}

func expandHome(p string) string { if p == "~" || strings.HasPrefix(p, "~/") { home, _ := os.UserHomeDir(); return filepath.Join(home, strings.TrimPrefix(p, "~/")) }; return p }
