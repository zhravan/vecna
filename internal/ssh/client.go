package ssh

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Session struct {
	client   *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
}

// Connect establishes an SSH session to h. If jumpHost is not nil, connects via that host as proxy.
// jumpSkipKey: when using jump host, set true to skip key auth for jump if key not yet deployed.
func Connect(h Host, password string, skipKeyIfNotDeployed bool, jumpHost *Host, jumpPassword string, jumpSkipKey bool) (*Session, error) {
	var client *ssh.Client
	var err error
	if jumpHost != nil {
		client, err = DialClientViaProxy(h, password, skipKeyIfNotDeployed, *jumpHost, jumpPassword, jumpSkipKey)
	} else {
		client, err = DialClient(h, password, skipKeyIfNotDeployed)
	}
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}

	if err := session.RequestPty(term, 80, 24, modes); err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to request pty: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start bash with no rc/profile so we get one clean window (no "logging into bash from zsh" etc).
	// User can run "source ~/.bashrc" or "bash -l" in-session if they want their full env.
	if err := session.Start("bash --norc --noprofile -i"); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	return &Session{
		client:   client,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
	}, nil
}

// DialClient establishes an SSH connection without starting a session. Used for port forwarding.
func DialClient(h Host, password string, skipKeyIfNotDeployed bool) (*ssh.Client, error) {
	return dialClient(h, password, skipKeyIfNotDeployed, nil, "")
}

// dialClient connects to h, optionally via proxyClient (jump host).
func dialClient(h Host, password string, skipKeyIfNotDeployed bool, proxyClient *ssh.Client, _ string) (*ssh.Client, error) {
	config, err := buildSSHConfig(h, password, skipKeyIfNotDeployed)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", h.Hostname, h.Port)

	if proxyClient == nil {
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			return nil, fmt.Errorf("failed to dial: %w", err)
		}
		return client, nil
	}

	conn, err := proxyClient.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target via proxy: %w", err)
	}
	clientConn, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to establish SSH to target: %w", err)
	}
	return ssh.NewClient(clientConn, chans, reqs), nil
}

// DialClientViaProxy connects to targetHost through jumpHost (bastion). Both use their own auth.
func DialClientViaProxy(targetHost Host, targetPassword string, targetSkipKey bool, jumpHost Host, jumpPassword string, jumpSkipKey bool) (*ssh.Client, error) {
	jumpClient, err := DialClient(jumpHost, jumpPassword, jumpSkipKey)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to jump host: %w", err)
	}
	client, err := dialClient(targetHost, targetPassword, targetSkipKey, jumpClient, jumpPassword)
	if err != nil {
		jumpClient.Close()
		return nil, err
	}
	// When client is closed, we should also close jumpClient. Wrap so closing the returned client closes the chain.
	// ssh.Client doesn't support wrapping Close. So we leave jumpClient open when target is closed (target conn is closed).
	// Optionally we could return a wrapper that closes both. For now we only close the target client; the jump stays open until process exits or we add a wrapper.
	return client, nil
}

// RunCommand connects to the host (optionally via jump), runs the command once, and returns combined stdout and stderr.
func RunCommand(h Host, password string, skipKey bool, jumpHost *Host, jumpPassword string, jumpSkipKey bool, command string) (output string, err error) {
	var client *ssh.Client
	if jumpHost != nil {
		client, err = DialClientViaProxy(h, password, skipKey, *jumpHost, jumpPassword, jumpSkipKey)
	} else {
		client, err = DialClient(h, password, skipKey)
	}
	if err != nil {
		return "", err
	}
	defer client.Close()
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.CombinedOutput(command)
	return string(out), err
}

func (s *Session) Resize(width, height int) error {
	return s.session.WindowChange(height, width)
}

func (s *Session) Write(data []byte) (int, error) {
	return s.stdin.Write(data)
}

func (s *Session) Read(p []byte) (int, error) {
	return s.stdout.Read(p)
}

func (s *Session) ReadError(p []byte) (int, error) {
	return s.stderr.Read(p)
}

func (s *Session) Close() error {
	if s.stdin != nil {
		s.stdin.Close()
	}
	if s.session != nil {
		s.session.Close()
	}
	if s.client != nil {
		return s.client.Close()
	}
	return nil
}

func buildSSHConfig(h Host, password string, skipKeyIfNotDeployed bool) (*ssh.ClientConfig, error) {
	home, _ := os.UserHomeDir()
	expandPath := func(p string) string {
		if strings.HasPrefix(p, "~") {
			return filepath.Join(home, p[1:])
		}
		return p
	}

	var authMethods []ssh.AuthMethod

	// If key is deployed, prioritize it; otherwise try password first
	keyAdded := false
	if h.IdentityFile != "" && !skipKeyIfNotDeployed {
		keyPath := expandPath(h.IdentityFile)
		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
				keyAdded = true
			}
		}
	}

	// Add password as fallback if key wasn't added, or as primary if no key
	if password != "" {
		if keyAdded {
			// Key first, then password as fallback
			authMethods = append(authMethods, ssh.Password(password))
		} else {
			// Password first if no key
			authMethods = append([]ssh.AuthMethod{ssh.Password(password)}, authMethods...)
		}
	}

	if len(authMethods) == 0 {
		keyPath := filepath.Join(home, ".ssh", "id_rsa")
		if key, err := os.ReadFile(keyPath); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		keyPath := filepath.Join(home, ".ssh", "id_ed25519")
		if key, err := os.ReadFile(keyPath); err == nil {
			if signer, err := ssh.ParsePrivateKey(key); err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            h.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	return config, nil
}

type Host struct {
	Name         string
	Hostname     string
	User         string
	Port         int
	IdentityFile string
}
