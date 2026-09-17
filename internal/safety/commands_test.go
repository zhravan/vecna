package safety

import "testing"

func TestIsDangerous(t *testing.T) { for _, cmd := range []string{"rm -rf /", "shutdown -r now", "mkfs.ext4 /dev/sda"} { if !IsDangerous(cmd) { t.Fatalf("expected dangerous: %q", cmd) } }; if IsDangerous("ls -la /tmp") { t.Fatal("safe command classified as dangerous") } }
