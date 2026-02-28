package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
)

// SSHConfigHost represents one host block from ~/.ssh/config, suitable for
// preview and mapping to vecna config.Host.
type SSHConfigHost struct {
	Name         string // first pattern (alias)
	Hostname     string
	User         string
	Port         int
	IdentityFile string // first IdentityFile if multiple
	ProxyJumpRaw string
}

// Load reads and parses path (typically ~/.ssh/config) and returns host
// entries. Skips the implicit "Host *" and blocks with no HostName.
// Returns error if the file cannot be read or parsed.
func Load(path string) ([]SSHConfigHost, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("home dir: %w", err)
		}
		path = filepath.Join(home, ".ssh", "config")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := ssh_config.DecodeBytes(b)
	if err != nil {
		return nil, fmt.Errorf("parse ssh config: %w", err)
	}

	var out []SSHConfigHost
	for _, host := range cfg.Hosts {
		if host.Patterns == nil || len(host.Patterns) == 0 {
			continue
		}
		alias := host.Patterns[0].String()
		alias = strings.TrimSpace(alias)
		// Skip implicit "Host *" or wildcard-only blocks
		if alias == "" || alias == "*" {
			continue
		}

		hostname, _ := cfg.Get(alias, "HostName")
		hostname = strings.TrimSpace(hostname)
		// Skip blocks with no HostName (e.g. some wildcard templates)
		if hostname == "" {
			continue
		}

		user, _ := cfg.Get(alias, "User")
		user = strings.TrimSpace(user)
		if user == "" {
			user = "root"
		}

		portStr, _ := cfg.Get(alias, "Port")
		port := 22
		if p := strings.TrimSpace(portStr); p != "" {
			if n, err := strconv.Atoi(p); err == nil && n > 0 && n < 65536 {
				port = n
			}
		}

		identityFiles, _ := cfg.GetAll(alias, "IdentityFile")
		identityFile := ""
		if len(identityFiles) > 0 {
			identityFile = strings.TrimSpace(identityFiles[0])
		}

		proxyJump, _ := cfg.Get(alias, "ProxyJump")
		proxyJump = strings.TrimSpace(proxyJump)

		out = append(out, SSHConfigHost{
			Name:         alias,
			Hostname:     hostname,
			User:         user,
			Port:         port,
			IdentityFile: identityFile,
			ProxyJumpRaw: proxyJump,
		})
	}

	return out, nil
}
