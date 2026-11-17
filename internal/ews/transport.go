// Пакет ews — read-only клиент Exchange Web Services (mailcat).
// Конверты операций — строго по docs/EWS-REFERENCE.md; TLS-изоляция и запрет
// секретов в логах — по docs/SECURITY.md.
package ews

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
)

// tlsVersion12 — минимально допустимая версия TLS (SECURITY.md §1).
const tlsVersion12 = tls.VersionTLS12

// newTLSConfig строит tls.Config для соединений с Exchange.
// Если caPath не пуст — доверие ТОЛЬКО приватному пулу из этого PEM-файла;
// системное хранилище macOS не используется. Если пуст — системные корни
// (публичный сертификат сервера). InsecureSkipVerify запрещён всегда.
func newTLSConfig(caPath string) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tlsVersion12}
	if caPath == "" {
		return cfg, nil
	}
	pem, err := os.ReadFile(caPath) // сам файл — 0600, см. SECURITY.md §6
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
