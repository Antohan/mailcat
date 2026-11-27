// Package ews — клиент Exchange Web Services (mailcat), только чтение.
// Конверты операций — строго по docs/EWS-REFERENCE.md; TLS-изоляция и запрет
// секретов в логах — по docs/SECURITY.md.
package ews

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"os"
	"time"
)

// TLSProbeError — результат проверки TLS для doctor: CertProblem отличает
// проблемы сертификата (нужно объяснение про ca_path) от прочих ошибок сети.
type TLSProbeError struct {
	CertProblem bool
	Err         error
}

func (e *TLSProbeError) Error() string { return e.Err.Error() }
func (e *TLSProbeError) Unwrap() error { return e.Err }

// TLSProbe — TLS-рукопожатие с изолированной конфигурацией из caPath
// (doctor, шаг 2; SECURITY.md §1). crypto/tls остаётся внутри пакета ews.
func TLSProbe(hostPort, caPath string) error {
	cfg, err := newTLSConfig(caPath)
	if err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", hostPort, cfg)
	if err != nil {
		var certErr *tls.CertificateVerificationError
		return &TLSProbeError{CertProblem: errors.As(err, &certErr), Err: err}
	}
	return conn.Close()
}

// tlsVersion12 — минимально допустимая версия TLS (SECURITY.md §1).
const tlsVersion12 = tls.VersionTLS12

// NewTLSConfig — экспорт конструктора изолированной TLS-конфигурации
// (используется doctor'ом для шага 2; SECURITY.md §1).
func NewTLSConfig(caPath string) (*tls.Config, error) { return newTLSConfig(caPath) }

// newTLSConfig строит tls.Config для соединений с Exchange.
// Если caPath не пуст — доверие ТОЛЬКО приватному пулу из этого PEM-файла;
// системное хранилище macOS не используется. Если пуст — системные корни
// (публичный сертификат сервера). InsecureSkipVerify запрещён всегда.
func newTLSConfig(caPath string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tlsVersion12}
	if caPath == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caPath) // #nosec G304 -- путь CA из конфига; права файла проверены (SECURITY.md §6)
	if err != nil {
		return nil, fmt.Errorf("чтение CA: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		return nil, errors.New("в файле CA нет валидных сертификатов PEM")
	}
	cfg.RootCAs = pool // заменяет системные корни ДЛЯ ЭТОГО КЛИЕНТА ТОЛЬКО
	return cfg, nil
}
