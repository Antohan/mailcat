package security

// Тесты безопасных имён файлов (docs/SECURITY.md §5): path traversal,
// фильтрация символов, длина, уникальность в каталоге.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeFileName(t *testing.T) {
	cases := []struct{ in, want string }{
		{"report.pdf", "report.pdf"},
		{"данные.csv", "______.csv"},   // кириллица вне фильтра [A-Za-z0-9._ -]
		{"../../etc/passwd", "passwd"}, // Base срезает весь путь
		{"/abs/path/x.exe", "x.exe"},
		{"a b c.txt", "a b c.txt"},
		{"weird|name*.txt?", "weird_name_.txt_"},
		{"", "attachment"},
		{".", "attachment"},
		{"..", "attachment"},
	}
	for _, tc := range cases {
		if got := SafeFileName(tc.in); got != tc.want {
			t.Errorf("SafeFileName(%q) = %q, хочу %q", tc.in, got, tc.want)
		}
	}
}

// TestSafeFileNameLength: длина ограничена 200 символами, расширение сохраняется.
func TestSafeFileNameLength(t *testing.T) {
	long := strings.Repeat("a", 300) + ".pdf"
	got := SafeFileName(long)
	if len(got) > 200 {
		t.Errorf("длина = %d, хочу ≤ 200", len(got))
	}
	if !strings.HasSuffix(got, ".pdf") {
		t.Errorf("расширение потеряно: %q", got)
	}
}

// TestUniquePath: конфликт имён — суффиксы (1), (2); свободное имя — как есть.
func TestUniquePath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "report.pdf"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := UniquePath(dir, "report.pdf"); got != filepath.Join(dir, "report (1).pdf") {
		t.Errorf("первый конфликт = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "report (1).pdf"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := UniquePath(dir, "report.pdf"); got != filepath.Join(dir, "report (2).pdf") {
		t.Errorf("второй конфликт = %q", got)
	}
	if got := UniquePath(dir, "free.txt"); got != filepath.Join(dir, "free.txt") {
		t.Errorf("свободное имя = %q", got)
	}
}

// TestSaveDirs: каталоги ~/Downloads/mailcat и временный создаются с правами 0700.
func TestSaveDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dl, err := DownloadDir()
	if err != nil {
		t.Fatalf("DownloadDir: %v", err)
	}
	if want := filepath.Join(home, "Downloads", "mailcat"); dl != want {
		t.Errorf("DownloadDir = %q, хочу %q", dl, want)
	}

	td, err := TempDir()
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}
	if !strings.Contains(td, "mailcat") {
		t.Errorf("TempDir = %q без mailcat", td)
	}

	for _, dir := range []string{dl, td} {
		st, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if st.Mode().Perm() != 0o700 {
			t.Errorf("права %s = %v, хочу 0700", dir, st.Mode().Perm())
		}
	}
}
