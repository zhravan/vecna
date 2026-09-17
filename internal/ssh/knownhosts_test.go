package ssh

import (
	"os"
	"path/filepath"
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

	key, err := ssh.NewPublicKey(&testSignerPublicKey{})
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

// testSignerPublicKey is only used to produce a deterministic ssh.PublicKey in tests.
type testSignerPublicKey struct{}

func (*testSignerPublicKey) Type() string { return "ssh-ed25519" }
func (*testSignerPublicKey) Marshal() []byte { return []byte{0, 0, 0, 11, 's', 's', 'h', '-', 'e', 'd', '2', '5', '5', '1', '9', 0, 0, 0, 0} }
