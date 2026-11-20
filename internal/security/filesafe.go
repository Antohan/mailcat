// Безопасные имена и каталоги для вложений (docs/SECURITY.md §5):
// filepath.Base + фильтр [A-Za-z0-9._ -], длина ≤ 200, конфликты — (1), (2);
// каталог ~/Downloads/mailcat и временный — с правами 0700.

package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxFileNameLen = 200

// allowedNameChars: A-Za-z0-9 точка подчёркивание пробел дефис; прочее — '_'.
func allowedNameRune(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return r
	case r == '.' || r == '_' || r == ' ' || r == '-':
		return r
	default:
		return '_'
	}
}

// SafeFileName превращает непроверенное имя вложения в безопасное для записи
// на диск: срезает пути (filepath.Base), фильтрует символы, ограничивает длину
// с сохранением расширения.
func SafeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || name == ".." || name == string(filepath.Separator) {
		return "attachment"
	}
	name = strings.Map(allowedNameRune, name)
	if len(name) > maxFileNameLen {
		ext := filepath.Ext(name)
		cut := maxFileNameLen - len(ext)
		if cut < 1 {
			cut = 1
			ext = ""
		}
		name = name[:cut] + ext
	}
	return name
}

// CreateUnique атомарно создаёт в dir новый файл под имя name (SafeFileName),
// правами 0600, и возвращает открытый на запись файл и итоговый путь; при
// конфликте имени — « (1)», « (2)»… перед расширением. Файл закрывает
// вызывающий.
// WHY: O_CREATE|O_EXCL занимает имя атомарно; прежний UniquePath
// «проверил os.Stat — записал потом» оставлял гонку между проверкой и записью.
// Не делать: вернуть только путь, а писать затем os.WriteFile — гонка вернётся.
func CreateUnique(dir, name string) (*os.File, string, error) {
	base := SafeFileName(name)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	for i := 0; ; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		path := filepath.Join(dir, candidate)
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return f, path, nil
		}
		if !os.IsExist(err) {
			return nil, "", fmt.Errorf("создание файла вложения %s: %w", path, err)
		}
	}
}

// DownloadDir возвращает (и создаёт с правами 0700) каталог сохранения
// вложений ~/Downloads/mailcat (SECURITY.md §5).
func DownloadDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("домашний каталог: %w", err)
	}
	dir := filepath.Join(home, "Downloads", "mailcat")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("каталог вложений %s: %w", dir, err)
	}
	return dir, nil
}

// TempDir возвращает (и создаёт с правами 0700) временный каталог mailcat
// для вложений, открываемых «на один раз» ($TMPDIR/mailcat). Очищается при
// выходе (этап 5).
func TempDir() (string, error) {
	base := os.TempDir()
	dir := filepath.Join(base, "mailcat")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("временный каталог вложений %s: %w", dir, err)
	}
	return dir, nil
}

// CleanupTempDir удаляет содержимое временного каталога вложений при выходе
// (PLAN.md этап 5). Сам каталог остаётся с правами 0700.
func CleanupTempDir() error {
	dir, err := TempDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("чтение временного каталога %s: %w", dir, err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return fmt.Errorf("очистка временного файла %s: %w", e.Name(), err)
		}
	}
	return nil
}
