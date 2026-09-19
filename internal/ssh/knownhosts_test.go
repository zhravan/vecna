package ssh

import (
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestKnownHostsPathUsesSSHDirectory(t *testing.T) {
	path := KnownHostsPath()
	if filepath.Base(path) != "known_hosts" {
		t.Fatalf("expected known_hosts filename, got %q", path)
	}
	if filepath.Base(filepath.Dir(path)) != ".ssh" {
		t.Fatalf("expected .ssh parent, got %q", filepath.Dir(path))
	}
}

func TestTrustHostKeyWritesOpenSSHEntry(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(private.Public())
	if err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("example.com:22", key); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected known_hosts entry")
	}
}

func TestTrustHostKeyIsIdempotent(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	_, private, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	key, err := ssh.NewPublicKey(private.Public())
	if err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("example.com:22", key); err != nil {
		t.Fatal(err)
	}
	if err := TrustHostKey("example.com:22", key); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".ssh", "known_hosts"))
	if err != nil {
		t.Fatal(err)
	}
	lines := 0
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) != "" {
			lines++
		}
	}
	if lines != 1 {
		t.Fatalf("expected one trusted entry, got %d", lines)
	}
}
