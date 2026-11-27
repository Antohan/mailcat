package main

// mailcat doctor — пошаговая диагностика подключения (PLAN.md этап 5,
// TESTING.md S1): 1) TCP-доступность (подсказка про VPN); 2) TLS-рукопожатие
// с изолированным пулом (объяснение про ca_path); 3) аутентификация;
// 4) FindFolder. Только read-only операции.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"time"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/security"
)

// doctorDeps — шаги диагностики, инжектируются в тестах. crypto/tls остаётся
// внутри internal/ews (policy_test): шаг 2 — ews.TLSProbe + TLSProbeError.
type doctorDeps struct {
	dialTCP      func(hostPort string) error
	tlsHandshake func(hostPort, caPath string) error
	findFolders  func(ctx context.Context, c *ews.Client) ([]ews.Folder, error)
	client       func(cfg *config.Config) (*ews.Client, error)
}

func realDoctorDeps(log *slog.Logger) doctorDeps {
	return doctorDeps{
		dialTCP: func(hostPort string) error {
			conn, err := net.DialTimeout("tcp", hostPort, 10*time.Second)
			if err != nil {
				return err
			}
			return conn.Close()
		},
		tlsHandshake: func(hostPort, caPath string) error {
			return ews.TLSProbe(hostPort, caPath)
		},
		client: func(cfg *config.Config) (*ews.Client, error) {
			pass, err := keychainPasswordFor(cfg)
			if err != nil {
				return nil, err
			}
			creds := ews.Credentials{Username: cfg.Username, Password: pass}
			return ews.NewClient(cfg.Server+"/EWS/Exchange.asmx", creds, config.ExpandHome(cfg.CAPath), log)
		},
		findFolders: func(ctx context.Context, c *ews.Client) ([]ews.Folder, error) {
			return c.FindFolders(ctx)
		},
	}
}

// hostPort извлекает хост:порт из server (https → 443 по умолчанию).
func hostPort(server string) string {
	u, err := url.Parse(server)
	if err != nil || u.Hostname() == "" {
		return server
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	return net.JoinHostPort(u.Hostname(), port)
}

// reporter пишет отчёт диагностики и запоминает первую ошибку записи:
// оборванный отчёт (закрытый stderr, полный диск) — это не «всё OK»,
// runDoctor в таком случае вернёт 1.
type reporter struct {
	w   io.Writer
	err error
}

// printf пишет строку отчёта; после первой ошибки записи молчит.
// shortcut: шаги не прерываем — их всего два запроса, проверка в каждом
// шаге дороже, чем доиграть сценарий, отчёт которого уже не прочитать.
func (r *reporter) printf(format string, a ...any) {
	if r.err != nil {
		return
	}
	_, r.err = fmt.Fprintf(r.w, format, a...)
}

// runDoctor выполняет 4 шага, пишет отчёт в w; код 0 — все шаги OK.
// Аутентификация (шаг 3) и FindFolder (шаг 4) — один запрос FindFolders;
// ошибка 401 относится к шагу 3, прочие — к шагу 4.
func runDoctor(w io.Writer, cfg *config.Config, deps doctorDeps) int {
	r := &reporter{w: w}
	code := r.run(cfg, deps)
	if r.err != nil {
		return 1
	}
	return code
}

func (r *reporter) run(cfg *config.Config, deps doctorDeps) int {
	hp := hostPort(cfg.Server)
	r.printf("mailcat doctor — диагностика подключения\nСервер: %s\n\n", hp)

	// Шаг 1: TCP.
	r.printf("[1/4] Соединение с %s … ", hp)
	if err := deps.dialTCP(hp); err != nil {
		r.printf("НЕТ\n      Сервер недоступен: %s\n", security.Sanitize(err.Error()))
		r.printf("      Похоже, VPN выключен — включите VPN и повторите (mailcat doctor).\n")
		return 1
	}
	r.printf("OK\n")

	// Шаг 2: TLS с изолированным пулом (crypto/tls — внутри ews.TLSProbe).
	r.printf("[2/4] TLS-рукопожатие (ca_path: %s) … ", caPathLabel(cfg.CAPath))
	if err := deps.tlsHandshake(hp, config.ExpandHome(cfg.CAPath)); err != nil {
		r.printf("ОШИБКА\n      %s\n", security.Sanitize(err.Error()))
		var probe *ews.TLSProbeError
		if errors.As(err, &probe) && probe.CertProblem {
			r.printf("      Сервер представил сертификат, который не прошёл проверку.\n" +
				"      Если сертификат корпоративный — укажите путь к root-CA (PEM) в ca_path;\n" +
				"      если ca_path уже задан — проверьте, что это правильный файл.\n" +
				"      Обход проверки сертификата в mailcat невозможен.\n")
		}
		return 1
	}
	r.printf("OK\n")

	// Шаг 3: сборка клиента с паролем из Keychain.
	client, err := deps.client(cfg)
	if err != nil {
		r.printf("[3/4] Аутентификация … ОШИБКА\n      %s\n", security.Sanitize(err.Error()))
		return 1
	}

	// Шаги 3 (сама проверка) + 4: запрос FindFolders.
	folders, err := deps.findFolders(context.Background(), client)
	switch {
	case err != nil && isAuthFailure(err):
		r.printf("[3/4] Аутентификация … ОШИБКА\n      %s\n", security.Sanitize(err.Error()))
		r.printf("      Проверьте логин (%s) и пароль.\n"+
			"      Пароль хранится в Keychain (запись mailcat); обновить:\n"+
			"        security delete-generic-password -s mailcat\n"+
			"        security add-generic-password -a %s -s mailcat -w\n", cfg.Username, cfg.Username)
		return 1
	case err != nil:
		r.printf("[3/4] Аутентификация … OK\n")
		r.printf("[4/4] Список папок … ОШИБКА\n      %s\n", security.Sanitize(err.Error()))
		if isAccessDenied(err) {
			r.printf("      Похоже, EWS отключён политикой для вашего ящика — спросите ИТ-отдел.\n")
		}
		return 1
	}

	r.printf("[3/4] Аутентификация … OK (пароль из Keychain, Basic/NTLM)\n")
	r.printf("[4/4] Список папок … OK (найдено папок: %d)\n", len(folders))
	for i, f := range folders {
		if i >= 10 {
			r.printf("      … и ещё %d\n", len(folders)-i)
			break
		}
		r.printf("      • %s (писем %d, непрочитанных %d)\n", security.Sanitize(f.DisplayName), f.TotalCount, f.UnreadCount)
	}
	return 0
}

func caPathLabel(p string) string {
	if p == "" {
		return "не задан — системные корни"
	}
	return p
}

func isAuthFailure(err error) bool {
	var httpErr *ews.HTTPError
	return errors.As(err, &httpErr) && httpErr.Status == 401
}

func isAccessDenied(err error) bool {
	var ewsErr *ews.ResponseError
	return errors.As(err, &ewsErr) && ewsErr.Code == "ErrorAccessDenied"
}
