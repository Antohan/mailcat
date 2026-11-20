package security

// Тесты-сторожа механических правил проекта (AGENTS.md, SECURITY.md).
// Они не проверяют поведение кода, а не дают нарушить read-only, TLS,
// зависимости и границы пакетов незаметно: `go test ./...` падает с
// русским сообщением, объясняющим, какое правило нарушено.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// forbiddenEWSOps — операции, которых не должно быть даже в исходниках:
// mailcat только читает (AGENTS.md, правило 1).
var forbiddenEWSOps = []string{
	"CreateItem", "UpdateItem", "DeleteItem", "SendItem", "MoveItem", "CopyItem",
	"MarkAsRead", "CreateFolder", "UpdateFolder", "DeleteFolder", "MoveFolder",
	"CopyFolder", "CreateAttachment", "DeleteAttachment", "SetItemField",
}

// allowedModules — одобренные прямые зависимости (AGENTS.md, таблица).
var allowedModules = []string{
	"github.com/rivo/tview",
	"github.com/gdamore/tcell/v2",
	"github.com/Azure/go-ntlmssp",
	"github.com/k3a/html2text",
	"golang.org/x/net",
	"github.com/BurntSushi/toml",
	"github.com/jhillyerd/enmime",
}

// importBoundaries — пакеты, которым разрешено тянуть чувствительные
// зависимости; ключ — импорт, значение — разрешённые каталоги.
var importBoundaries = map[string][]string{
	"net/http":                    {"internal/ews"},
	"crypto/tls":                  {"internal/ews"},
	"os/exec":                     {"internal/platform", "cmd/mailcat"},
	"github.com/rivo/tview":       {"internal/ui"},
	"github.com/Azure/go-ntlmssp": {"internal/ews"},
}

func TestReadOnlyEWSOperations(t *testing.T) {
	root := repoRoot(t)
	seen := map[string]bool{}
	for _, path := range sourceFiles(t, root, false) {
		rel := relToRoot(root, path)
		f := parsedFile(t, path)
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				return true
			}
			for _, op := range forbiddenEWSOps {
				if !strings.Contains(s, op) {
					continue
				}
				key := rel + ":" + op
				if seen[key] {
					continue
				}
				seen[key] = true
				t.Errorf("%s: запрещённая EWS-операция %q — клиент только читает (AGENTS.md, правило 1)", rel, op)
			}
			return true
		})
	}

	testdata := filepath.Join(root, "internal", "ews", "testdata")
	entries, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatalf("чтение testdata: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(testdata, e.Name()))
		if err != nil {
			t.Fatalf("чтение %s: %v", e.Name(), err)
		}
		for _, op := range forbiddenEWSOps {
			if strings.Contains(string(data), op) {
				t.Errorf("internal/ews/testdata/%s: запрещённая EWS-операция %q (AGENTS.md, правило 1)", e.Name(), op)
			}
		}
	}
}

func TestNoInsecureSkipVerify(t *testing.T) {
	root := repoRoot(t)
	for _, path := range sourceFiles(t, root, false) {
		f := parsedFile(t, path)
		ast.Inspect(f, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if ok && ident.Name == "InsecureSkipVerify" {
				t.Errorf("%s: InsecureSkipVerify запрещён даже для отладки (SECURITY.md §1)", relToRoot(root, path))
			}
			return true
		})
	}
}

func TestDependenciesAreApproved(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "go.mod"))
	if err != nil {
		t.Fatalf("чтение go.mod: %v", err)
	}
	for _, mod := range directRequires(string(data)) {
		if !slices.Contains(allowedModules, mod) {
			t.Errorf("go.mod: новая зависимость %q — сначала спроси пользователя (AGENTS.md, правило 6)", mod)
		}
	}
}

func TestDirectRequires(t *testing.T) {
	gomod := `module mailcat

go 1.26.0

require github.com/Azure/go-ntlmssp v0.1.1

require (
	github.com/rivo/tview v0.42.0
	golang.org/x/net v0.62.0 // indirect
)

require github.com/BurntSushi/toml v1.6.0 // indirect
`
	want := []string{"github.com/Azure/go-ntlmssp", "github.com/rivo/tview"}
	if got := directRequires(gomod); !slices.Equal(got, want) {
		t.Fatalf("directRequires = %v, хотим %v", got, want)
	}
}

func TestImportBoundaries(t *testing.T) {
	root := repoRoot(t)
	for _, path := range sourceFiles(t, root, false) {
		rel := relToRoot(root, path)
		f := parsedFile(t, path)
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				continue
			}
			allowedDirs, restricted := importBoundaries[p]
			if !restricted {
				continue
			}
			allowedHere := false
			for _, dir := range allowedDirs {
				if rel == dir || strings.HasPrefix(rel, dir+"/") {
					allowedHere = true
					break
				}
			}
			if !allowedHere {
				t.Errorf("%s: импорт %q разрешён только в %s (SECURITY.md §4.5, §7)",
					rel, p, strings.Join(allowedDirs, ", "))
			}
		}
	}
}

func TestNoPasswordInConfig(t *testing.T) {
	root := repoRoot(t)

	examplePath := filepath.Join(root, "config.example.toml")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("чтение config.example.toml: %v", err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		key := strings.TrimSpace(strings.SplitN(trimmed, "=", 2)[0])
		if strings.EqualFold(key, "password") {
			t.Errorf("config.example.toml:%d: пароль хранится только в Keychain (SECURITY.md §2)", i+1)
		}
	}

	for _, path := range sourceFiles(t, root, false) {
		rel := relToRoot(root, path)
		if !strings.HasPrefix(rel, "internal/config/") {
			continue
		}
		f := parsedFile(t, path)
		ast.Inspect(f, func(n ast.Node) bool {
			st, ok := n.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if strings.Contains(strings.ToLower(name.Name), "password") {
						t.Errorf("%s: поле %s — пароль хранится только в Keychain (SECURITY.md §2)", rel, name.Name)
					}
				}
				if field.Tag == nil {
					continue
				}
				tag, err := strconv.Unquote(field.Tag.Value)
				if err == nil && strings.Contains(strings.ToLower(tag), "password") {
					t.Errorf("%s: toml-тег %q — пароль хранится только в Keychain (SECURITY.md §2)", rel, tag)
				}
			}
			return true
		})
	}
}

// directRequires возвращает прямые зависимости из go.mod, пропуская
// // indirect: косвенные пакеты тянут одобренные библиотеки, их не проверяем.
func directRequires(gomod string) []string {
	var mods []string
	inBlock := false
	for _, raw := range strings.Split(gomod, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "require ("):
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		}
		if line == "" || strings.HasPrefix(line, "//") || strings.Contains(line, "// indirect") {
			continue
		}
		fields := strings.Fields(line)
		switch {
		case inBlock && len(fields) > 0:
			mods = append(mods, fields[0])
		case strings.HasPrefix(line, "require ") && len(fields) > 1:
			mods = append(mods, fields[1])
		}
	}
	return mods
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("не удалось определить путь к policy_test.go")
	}
	// internal/security/policy_test.go → корень модуля.
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

func relToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// sourceFiles обходит Go-файлы репозитория, пропуская служебные каталоги.
func sourceFiles(t *testing.T, root string, withTests bool) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "testdata", ".opencode":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !withTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		t.Fatalf("обход репозитория: %v", err)
	}
	slices.Sort(files)
	return files
}

func parsedFile(t *testing.T, path string) *ast.File {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("разбор %s: %v", path, err)
	}
	return f
}
