package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCleanupTempDir: временные копии вложений удаляются при выходе
// (PLAN.md этап 5); сам каталог остаётся (права 0700 сохраняются).
func TestCleanupTempDir(t *testing.T) {
	dir, err := TempDir()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	junk := filepath.Join(dir, "report.pdf")
	if err := os.WriteFile(junk, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := CleanupTempDir(); err != nil {
		t.Fatalf("CleanupTempDir: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("временный каталог не пуст: %v", entries)
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Errorf("права каталога = %v, хочу 0700", st.Mode().Perm())
	}
}
