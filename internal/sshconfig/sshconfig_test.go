package sshconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadExpandsIncludesAndAliasOnlyHosts(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "config")
	inc := filepath.Join(dir, "conf.d", "work")
	if err := os.MkdirAll(filepath.Dir(inc), 0700); err != nil { t.Fatal(err) }
	if err := os.WriteFile(inc, []byte("Host app\n    User deploy\n    Port 2222\n"), 0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(main, []byte("Include conf.d/*\n"), 0600); err != nil { t.Fatal(err) }
	hosts, err := Load(main)
	if err != nil { t.Fatal(err) }
	if len(hosts) != 1 { t.Fatalf("expected one host, got %d", len(hosts)) }
	if hosts[0].Name != "app" || hosts[0].Hostname != "app" || hosts[0].User != "deploy" || hosts[0].Port != 2222 { t.Fatalf("unexpected host: %+v", hosts[0]) }
}
