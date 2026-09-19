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
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if seen[abs] {
		return nil, nil
	}
	seen[abs] = true
	b, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	base := filepath.Dir(abs)
	var out strings.Builder
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		fields := strings.Fields(trim)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Include") {
			for _, pattern := range fields[1:] {
				if strings.HasPrefix(pattern, "~") {
					pattern = expandHome(pattern)
				} else if !filepath.IsAbs(pattern) {
					pattern = filepath.Join(base, pattern)
				}
				matches, _ := filepath.Glob(pattern)
				for _, match := range matches {
					child, e := expandFile(match, seen)
					if e != nil {
						return nil, fmt.Errorf("include %s: %w", match, e)
					}
					out.Write(child)
				}
			}
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return []byte(out.String()), nil
}

// UpsertAlias writes a Vecna host into OpenSSH config without touching other
// aliases. Existing exact Host blocks are replaced; Include directives remain.
func UpsertAlias(path string, host SSHConfigHost) error {
	path = expandHome(path)
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, ".ssh", "config")
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		b = nil
	} else if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")
	var out []string
	replaced := false
	for i := 0; i < len(lines); {
		fields := strings.Fields(strings.TrimSpace(lines[i]))
		if len(fields) >= 2 && strings.EqualFold(fields[0], "Host") && fields[1] == host.Name {
			if !replaced {
				out = append(out, renderHost(host)...)
				replaced = true
			}
			i++
			for i < len(lines) {
				f := strings.Fields(strings.TrimSpace(lines[i]))
				if len(f) >= 2 && strings.EqualFold(f[0], "Host") {
					break
				}
				i++
			}
			continue
		}
		out = append(out, lines[i])
		i++
	}
	if !replaced {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, renderHost(host)...)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0600)
}
func renderHost(h SSHConfigHost) []string {
	lines := []string{"Host " + h.Name, "    HostName " + h.Hostname, "    User " + h.User, fmt.Sprintf("    Port %d", h.Port)}
	if h.IdentityFile != "" {
		lines = append(lines, "    IdentityFile "+h.IdentityFile)
	}
	if h.ProxyJumpRaw != "" {
		lines = append(lines, "    ProxyJump "+h.ProxyJumpRaw)
	}
	return append(lines, "")
}
func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, strings.TrimPrefix(p, "~/"))
	}
	return p
}
