// Команда mailcat — входная точка: подкоманды run (TUI), doctor (диагностика,
// этап 5), folders (список папок в stdout — дымовый тест EWS).
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/mail"
	"mailcat/internal/platform"
	"mailcat/internal/security"
	"mailcat/internal/ui"
)

// runtime — зависимости команды: подменяются в тестах.
type runtime struct {
	keychainPassword func() (string, error) // Keychain macOS
	newUI            func(cfg *config.Config, c *ews.Client) error
	findFolders      func(ctx context.Context, c *ews.Client) ([]ews.Folder, error)
}

func mainRuntime() runtime {
	return runtime{
		keychainPassword: platform.KeychainPassword,
		newUI: func(cfg *config.Config, c *ews.Client) error {
			return ui.New(cfg, c).Run()
		},
		findFolders: func(ctx context.Context, c *ews.Client) ([]ews.Folder, error) {
			return c.FindFolders(ctx)
		},
	}
}

func main() {
	osExit(run(os.Args[1:], os.Stdout, os.Stderr, mainRuntime()))
}

// osExit — переменная для тестов (подменять exit-код без os.Exit).
var osExit = os.Exit

// run обрабатывает аргументы и возвращает код выхода; stdout/stderr передаются
// явно, чтобы работать без запуска процесса в тестах.
func run(argv []string, stdout, stderr io.Writer, rt runtime) int {
	if len(argv) == 0 {
		fmt.Fprintln(stderr, "не указана команда")
		usage(stderr)
		return 2
	}
	switch argv[0] {
	case "run", "doctor", "folders":
		// см. ниже
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "неизвестная команда: %s\n", argv[0])
		usage(stderr)
		return 2
	}

	path := config.DefaultPath()
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "Не найден конфигурационный файл: %s\n"+
				"Скопируйте config.example.toml в этот путь и заполните свои значения.\n", path)
			return 1
		}
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(stderr, "Конфигурация некорректна: %v\n", err)
		return 1
	}

	log, closer, lerr := platform.NewLogger(cfg.Log.File)
	if lerr != nil {
		log, closer, _ = platform.NewLogger("")
		fmt.Fprintf(stderr, "внимание: файловый лог недоступен: %v\n", lerr)
	}
	defer closer()

	// В лог — только хост сервера: SECURITY.md §2 (без учётных данных).
	host := cfg.Server
	if u, perr := url.Parse(cfg.Server); perr == nil && u.Host != "" {
		host = u.Host
	}
	log.Info("mailcat запуск", "cmd", argv[0], "server", host)

	switch argv[0] {
	case "doctor":
		fmt.Fprintln(stderr, "Диагностика будет реализована на этапе 5 разработки.")
		return 1
	case "folders":
		return runFolders(stdout, stderr, cfg, rt)
	default: // run
		return runTUI(stderr, cfg, rt)
	}
}

// buildClient собирает EWS-клиент: endpoint server + /EWS/Exchange.asmx
// (EWS-REFERENCE.md §1), пароль из Keychain, CA из ca_path.
func buildClient(cfg *config.Config, rt runtime) (*ews.Client, error) {
	pass, err := rt.keychainPassword()
	if err != nil {
		return nil, err
	}
	creds := ews.Credentials{Username: cfg.Username, Password: pass}
	return ews.NewClient(cfg.Server+"/EWS/Exchange.asmx", creds, config.ExpandHome(cfg.CAPath), nil)
}

// runTUI запускает интерфейс почты.
func runTUI(stderr io.Writer, cfg *config.Config, rt runtime) int {
	client, err := buildClient(cfg, rt)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	if err := rt.newUI(cfg, client); err != nil {
		fmt.Fprintf(stderr, "ошибка интерфейса: %v\n", err)
		return 1
	}
	return 0
}

// runFolders печатает список папок с непрочитанными — дымовый тест EWS
// (PLAN.md этап 1, TESTING.md S2). Весь текст из EWS — через санитайзер.
func runFolders(stdout, stderr io.Writer, cfg *config.Config, rt runtime) int {
	client, err := buildClient(cfg, rt)
	if err != nil {
		fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}
	list, err := rt.findFolders(context.Background(), client)
	if err != nil {
		fmt.Fprintf(stderr, "%s\n", security.Sanitize(err.Error()))
		return 1
	}
	folders := mail.Folders(list)
	for _, f := range folders {
		fmt.Fprintf(stdout, "%s — писем: %d, непрочитанных: %d\n",
			security.Sanitize(f.DisplayName), f.TotalCount, f.UnreadCount)
	}
	return 0
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Использование: mailcat <команда>")
	fmt.Fprintln(w, "Команды:")
	fmt.Fprintln(w, "  run     — почтовый TUI")
	fmt.Fprintln(w, "  doctor  — пошаговая диагностика подключения")
	fmt.Fprintln(w, "  folders — список папок в stdout (дымовый тест EWS)")
}
