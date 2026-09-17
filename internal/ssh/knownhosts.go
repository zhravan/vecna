package ssh

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var errHostKeyProbe = errors.New("vecna: host key probe complete")

type UnknownHostKeyError struct {
	Host        string
	Fingerprint string
	Algorithm   string
	Key         ssh.PublicKey
}

func (e *UnknownHostKeyError) Error() string {
	return fmt.Sprintf("unknown SSH host key for %s (fingerprint %s)", e.Host, e.Fingerprint)
}

type ChangedHostKeyError struct {
	Host                 string
	Fingerprint          string
	Algorithm            string
	Key                  ssh.PublicKey
	PreviousFingerprints []string
	Cause                error
}

func (e *ChangedHostKeyError) Error() string {
	return fmt.Sprintf("SSH host key changed for %s (presented fingerprint %s)", e.Host, e.Fingerprint)
}

func (e *ChangedHostKeyError) Unwrap() error { return e.Cause }

var probeMu sync.Mutex

func KnownHostsPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".ssh", "known_hosts")
	}
	return filepath.Join(home, ".ssh", "known_hosts")
}

func ensureKnownHostsFile() (string, error) {
	path := KnownHostsPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("create SSH directory: %w", err)
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return "", fmt.Errorf("create known_hosts: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("close known_hosts: %w", err)
		}
	} else if err != nil {
		return "", fmt.Errorf("stat known_hosts: %w", err)
	}
	return path, nil
}

func HostKeyCallback() (ssh.HostKeyCallback, error) {
	path, err := ensureKnownHostsFile()
	if err != nil {
		return nil, err
	}
	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts: %w", err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		if err := callback(hostname, remote, key); err != nil {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				fingerprint := ssh.FingerprintSHA256(key)
				if len(keyErr.Want) == 0 {
					return &UnknownHostKeyError{
						Host:        hostname,
						Fingerprint: fingerprint,
						Algorithm:   key.Type(),
						Key:         key,
					}
				}
				previous := make([]string, 0, len(keyErr.Want))
				for _, want := range keyErr.Want {
					if want.Key != nil {
						previous = append(previous, ssh.FingerprintSHA256(want.Key))
					}
				}
				return &ChangedHostKeyError{
					Host:                 hostname,
					Fingerprint:          fingerprint,
					Algorithm:            key.Type(),
					Key:                  key,
					PreviousFingerprints: previous,
					Cause:                err,
				}
			}
			return err
		}
		return nil
	}, nil
}

func TrustHostKey(host string, key ssh.PublicKey) error {
	if key == nil {
		return fmt.Errorf("cannot trust empty host key")
	}
	path, err := ensureKnownHostsFile()
	if err != nil {
		return err
	}
	line := knownhosts.Line([]string{knownhosts.Normalize(host)}, key)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open known_hosts: %w", err)
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, line); err != nil {
		return fmt.Errorf("write known_hosts: %w", err)
	}
	return nil
}

func ProbeHostKey(host Host) (fingerprint, algorithm string, key ssh.PublicKey, err error) {
	probeMu.Lock()
	defer probeMu.Unlock()

	port := host.Port
	if port == 0 {
		port = 22
	}
	addr := fmt.Sprintf("%s:%d", host.Hostname, port)
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return "", "", nil, fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer conn.Close()

	var presented ssh.PublicKey
	cfg := &ssh.ClientConfig{
		User: host.User,
		HostKeyCallback: func(_ string, _ net.Addr, k ssh.PublicKey) error {
			presented = k
			return errHostKeyProbe
		},
		Timeout: 5 * time.Second,
	}
	_, _, _, handshakeErr := ssh.NewClientConn(conn, addr, cfg)
	if presented == nil {
		if handshakeErr != nil {
			return "", "", nil, fmt.Errorf("probe host key: %w", handshakeErr)
		}
		return "", "", nil, fmt.Errorf("probe host key: server did not present a host key")
	}
	return ssh.FingerprintSHA256(presented), presented.Type(), presented, nil
}
