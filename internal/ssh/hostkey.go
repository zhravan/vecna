package ssh

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyError is returned when a server key is not yet trusted or no longer
// matches the user's known_hosts file.
type HostKeyError struct {
	Host        string
	Address     string
	Fingerprint string
	Key         ssh.PublicKey
	FirstUse    bool
	Err         error
}

func (e *HostKeyError) Error() string {
	if e.FirstUse {
		return fmt.Sprintf("host key for %s is not trusted (fingerprint: %s)", e.Host, e.Fingerprint)
	}
	return fmt.Sprintf("host key verification failed for %s (fingerprint: %s): %v", e.Host, e.Fingerprint, e.Err)
}

func KnownHostsPath() string {
	if p := strings.TrimSpace(os.Getenv("VECNA_KNOWN_HOSTS")); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".ssh", "known_hosts")
}

func ensureKnownHostsFile() error {
	path := KnownHostsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	return f.Close()
}

func hostKeyCallback(host Host) (ssh.HostKeyCallback, error) {
	if err := ensureKnownHostsFile(); err != nil {
		return nil, fmt.Errorf("prepare known_hosts: %w", err)
	}
	callback, err := knownhosts.New(KnownHostsPath())
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := callback(hostname, remote, key); err == nil {
			return nil
		} else {
			firstUse := false
			if ke, ok := err.(*knownhosts.KeyError); ok {
				firstUse = len(ke.Want) == 0 && ke.Offending == nil
			}
			return &HostKeyError{
				Host:        host.Name,
				Address:     hostname,
				Fingerprint: ssh.FingerprintSHA256(key),
				Key:         key,
				FirstUse:    firstUse,
				Err:         err,
			}
		}
	}, nil
}

func hostPattern(host Host) string {
	if host.Port == 0 || host.Port == 22 {
		return host.Hostname
	}
	return net.JoinHostPort(host.Hostname, fmt.Sprintf("%d", host.Port))
}

// TrustHostKey records a key after the caller has explicitly confirmed its
// fingerprint. Existing entries are intentionally never replaced here.
func TrustHostKey(host Host, key ssh.PublicKey) error {
	if key == nil {
		return fmt.Errorf("cannot trust an empty host key")
	}
	if err := ensureKnownHostsFile(); err != nil {
		return err
	}
	path := KnownHostsPath()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	line := fmt.Sprintf("%s %s\n", hostPattern(host), strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))))
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write known_hosts: %w", err)
	}
	return nil
}

// ProbeHostKey obtains the server public key without making a trust decision.
// Callers must display the fingerprint and obtain explicit confirmation before
// calling TrustHostKey.
func ProbeHostKey(host Host) (ssh.PublicKey, error) {
	var captured ssh.PublicKey
	callback := func(_ string, _ net.Addr, key ssh.PublicKey) error {
		captured = key
		return nil
	}
	config := &ssh.ClientConfig{
		User:            host.User,
		HostKeyCallback: callback,
		Timeout:         5 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", host.Hostname, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, err
	}
	_ = client.Close()
	return captured, nil
}
