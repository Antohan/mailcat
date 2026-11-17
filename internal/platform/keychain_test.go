package platform

import (
	"testing"
)

// TestKeychainCmd: команда читает пароль сервиса mailcat из Keychain macOS
// (SECURITY.md §2: security find-generic-password -s mailcat -w).
func TestKeychainCmd(t *testing.T) {
	cmd := keychainCmd()
	// exec резолвит "security" через PATH в полный путь macOS.
	if cmd.Path != "/usr/bin/security" {
		t.Errorf("Path = %q, хочу /usr/bin/security", cmd.Path)
	}
	want := []string{"security", "find-generic-password", "-s", "mailcat", "-w"}
	if len(cmd.Args) != len(want) {
		t.Fatalf("Args = %v, хочу %v", cmd.Args, want)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Errorf("Args[%d] = %q, хочу %q", i, cmd.Args[i], want[i])
		}
	}
}
