package main

// mailcat doctor — пошаговая диагностика подключения (PLAN.md этап 5,
// TESTING.md S1): 1) TCP-доступность (подсказка про VPN); 2) TLS-рукопожатие
// с изолированным пулом (объяснение про ca_path); 3) аутентификация;
// 4) FindFolder. Только read-only операции.

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"mailcat/internal/config"
	"mailcat/internal/ews"
	"mailcat/internal/security"
)

// doctorDeps — шаги диагностики, инжектируются в тестах.
type doctorDeps struct {
	dialTCP      func(hostPort string) error
	tlsHandshake func(hostPort string, tlsCfg *tls.Config) error
	findFolders  func(ctx context.Context, c *ews.Client) ([]ews.Folder, error)
	client       func(cfg *config.Config) (*ews.Client, error)
}

func realDoctorDeps() doctorDeps {
	return doctorDeps{
		dialTCP: func(hostPort string) error {
			conn, err := net.DialTimeout("tcp", hostPort, 10*time.Second)
			if err != nil {
				return err
			}
			return conn.Close()
		},
		tlsHandshake: func(hostPort string, tlsCfg *tls.Config) error {
			dialer := &net.Dialer{Timeout: 10 * time.Second}
			conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, tlsCfg)
			if err != nil {
				return err
			}
			return conn.Close()
		},
		client: func(cfg *config.Config) (*ews.Client, error) {
			pass, err := keychainPasswordFor(cfg)
			if err != nil {
				return nil, err
			}
			creds := ews.Credentials{Username: cfg.Username, Password: pass}
			return ews.NewClient(cfg.Server+"/EWS/Exchange.asmx", creds, config.ExpandHome(cfg.CAPath), nil)
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

// runDoctor выполняет 4 шага, пишет отчёт в w; код 0 — все шаги OK.
// Аутентификация (шаг 3) и FindFolder (шаг 4) — один запрос FindFolders;
// ошибка 401 относится к шагу 3, прочие — к шагу 4.
func runDoctor(w io.Writer, cfg *config.Config, deps doctorDeps) int {
	hp := hostPort(cfg.Server)
	fmt.Fprintf(w, "mailcat doctor — диагностика подключения\nСервер: %s\n\n", hp)

	// Шаг 1: TCP.
	fmt.Fprintf(w, "[1/4] Соединение с %s … ", hp)
	if err := deps.dialTCP(hp); err != nil {
		fmt.Fprintf(w, "НЕТ\n      Сервер недоступен: %v\n", err)
		fmt.Fprintf(w, "      Похоже, VPN выключен — включите VPN и повторите (mailcat doctor).\n")
		return 1
	}
	fmt.Fprintln(w, "OK")

	// Шаг 2: TLS с изолированным пулом.
	fmt.Fprintf(w, "[2/4] TLS-рукопожатие (ca_path: %s) … ", caPathLabel(cfg.CAPath))
	tlsCfg, err := ews.NewTLSConfig(config.ExpandHome(cfg.CAPath))
	if err != nil {
		fmt.Fprintf(w, "ОШИБКА\n      %v\n", err)
		return 1
	}
	if err := deps.tlsHandshake(hp, tlsCfg); err != nil {
		fmt.Fprintf(w, "ОШИБКА\n      %v\n", err)
		var certErr *tls.CertificateVerificationError
		if errors.As(err, &certErr) {
			fmt.Fprintf(w, "      Сервер представил сертификат, который не прошёл проверку.\n"+
				"      Если сертификат корпоративный — укажите путь к root-CA (PEM) в ca_path;\n"+
				"      если ca_path уже задан — проверьте, что это правильный файл.\n"+
				"      Обход проверки сертификата в mailcat невозможен.\n")
		}
		return 1
	}
	fmt.Fprintln(w, "OK")

	// Шаг 3: сборка клиента с паролем из Keychain.
	client, err := deps.client(cfg)
	if err != nil {
		fmt.Fprintf(w, "[3/4] Аутентификация … ОШИБКА\n      %v\n", err)
		return 1
	}

	// Шаги 3 (сама проверка) + 4: запрос FindFolders.
	folders, err := deps.findFolders(context.Background(), client)
	switch {
	case err != nil && isAuthFailure(err):
		fmt.Fprintf(w, "[3/4] Аутентификация … ОШИБКА\n      %v\n", err)
		fmt.Fprintf(w, "      Проверьте логин (%s) и пароль.\n"+
			"      Пароль хранится в Keychain (запись mailcat); обновить:\n"+
			"        security delete-generic-password -s mailcat\n"+
			"        security add-generic-password -a %s -s mailcat -w\n", cfg.Username, cfg.Username)
		return 1
	case err != nil:
		fmt.Fprint(w, "[3/4] Аутентификация … OK\n")
		fmt.Fprintf(w, "[4/4] Список папок … ОШИБКА\n      %v\n", err)
		if isAccessDenied(err) {
			fmt.Fprintf(w, "      Похоже, EWS отключён политикой для вашего ящика — спросите ИТ-отдел.\n")
		}
		return 1
	}

	fmt.Fprint(w, "[3/4] Аутентификация … OK (пароль из Keychain, Basic/NTLM)\n")
	fmt.Fprintf(w, "[4/4] Список папок … OK (найдено папок: %d)\n", len(folders))
	for i, f := range folders {
		if i >= 10 {
			fmt.Fprintf(w, "      … и ещё %d\n", len(folders)-i)
			break
		}
		fmt.Fprintf(w, "      • %s (писем %d, непрочитанных %d)\n", security.Sanitize(f.DisplayName), f.TotalCount, f.UnreadCount)
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
	var ewsErr *ews.EWSError
	return errors.As(err, &ewsErr) && ewsErr.Code == "ErrorAccessDenied"
}
