package tui

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
 )

func normalizeDroppedPaths(payload string) []string {
	payload = strings.TrimSpace(payload)
	if payload == "" { return nil }
	if p, ok := normalizeDroppedPath(payload); ok { return []string{p} }
	tokens, ok := splitDroppedPathTokens(payload)
	if !ok || len(tokens) == 0 { return nil }
	paths := make([]string, 0, len(tokens))
	for _, token := range tokens {
		p, ok := normalizeDroppedPath(token)
		if !ok { return nil }
		paths = append(paths, p)
	}
	return paths
}

func normalizeDroppedPath(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" { return "", false }
	if strings.HasPrefix(raw, "file://") {
		u, err := url.Parse(raw)
		if err != nil || u.Path == "" { return "", false }
		raw = u.Path
	}
	if len(raw) >= 2 && ((raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'')) {
		if unquoted, err := strconv.Unquote(raw); err == nil { raw = unquoted } else { raw = raw[1:len(raw)-1] }
	}
	raw = strings.ReplaceAll(raw, "\\\\ ", " ")
	raw = strings.ReplaceAll(raw, "\\\\'", "'")
	raw = strings.ReplaceAll(raw, "\\\\\"", "\"")
	raw = strings.TrimSpace(raw)
	if raw == "" { return "", false }
	if strings.HasPrefix(raw, "~/") || strings.HasPrefix(raw, "~\\") {
		home, err := os.UserHomeDir()
		if err != nil { return "", false }
		raw = filepath.Join(home, raw[2:])
	}
	if !filepath.IsAbs(raw) { return "", false }
	abs, err := filepath.Abs(raw)
	if err != nil { return "", false }
	if _, err := os.Stat(abs); err != nil { return "", false }
	return abs, true
}

func splitDroppedPathTokens(payload string) ([]string, bool) {
	var tokens []string
	var b strings.Builder
	var quote rune
	escaped := false
	flush := func() { if b.Len() > 0 { tokens = append(tokens, b.String()); b.Reset() } }
	for _, r := range payload {
		if escaped { b.WriteRune(r); escaped = false; continue }
		if r == '\\' && quote != '\'' { escaped = true; continue }
		if quote != 0 { if r == quote { quote = 0 } else { b.WriteRune(r) }; continue }
		switch r {
		case '\'', '"': quote = r
		case ' ', '\t', '\r', '\n': flush()
		default: b.WriteRune(r)
		}
	}
	if escaped { b.WriteRune('\\') }
	if quote != 0 { return nil, false }
	flush()
	return tokens, true
}