package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOpenCmd: команда открывает путь через /usr/bin/open (macOS).
func TestOpenCmd(t *testing.T) {
	cmd := openCmd("/tmp/report.pdf")
	if cmd.Path != "/usr/bin/open" {
		t.Errorf("Path = %q, хочу /usr/bin/open", cmd.Path)
	}
	if len(cmd.Args) != 2 || cmd.Args[0] != "/usr/bin/open" || cmd.Args[1] != "/tmp/report.pdf" {
		t.Errorf("Args = %v, хочу [/usr/bin/open /tmp/report.pdf]", cmd.Args)
	}
}

// TestNewLoggerWrites: логгер создаёт каталоги и файл, записи попадают в файл.
func TestNewLoggerWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state", "mailcat", "mailcat.log")
	log, closer, err := NewLogger(path)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	log.Info("запуск mailcat")
	closer()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("чтение лога: %v", err)
	}
	if !strings.Contains(string(data), "запуск mailcat") {
		t.Errorf("в логе нет сообщения: %q", data)
	}
}

// TestNewLoggerPerms: каталог лога 0700, файл 0600 (SECURITY.md §6).
func TestNewLoggerPerms(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "mailcat.log")
	_, closer, err := NewLogger(path)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}
	closer()

	st, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o700 {
		t.Errorf("права каталога лога = %v, хочу 0700", st.Mode().Perm())
	}
	st, err = os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("права файла лога = %v, хочу 0600", st.Mode().Perm())
	}
}

// TestNewLoggerDiscard: пустой путь — логгер в никуда, запись не падает.
func TestNewLoggerDiscard(t *testing.T) {
	log, closer, err := NewLogger("")
	if err != nil {
		t.Fatalf("NewLogger(\"\"): %v", err)
	}
	if closer == nil {
		t.Fatal("closer = nil")
	}
	log.Info("никуда")
	closer()
}
