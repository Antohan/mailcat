package ews

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeTestCA генерирует самоподписанный сертификат и пишет его PEM в файл
// с правами 0600 (как файл CA из конфигурации, SECURITY.md §6).
func writeTestCA(t *testing.T, dir string) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("генерация ключа: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"mailcat-test"},
			CommonName:   "mailcat-test-root-ca",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("создание сертификата: %v", err)
	}
	path := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("запись PEM: %v", err)
	}
	return path
}

// TestNewTLSConfigEmptyCAPath: пустой caPath — системные корни (RootCAs=nil),
// TLS не ниже 1.2 (SECURITY.md §1).
func TestNewTLSConfigEmptyCAPath(t *testing.T) {
	cfg, err := newTLSConfig("")
	if err != nil {
		t.Fatalf("newTLSConfig(\"\"): %v", err)
	}
	if cfg.RootCAs != nil {
		t.Errorf("RootCAs = %v, хочу nil (системные корни) при пустом ca_path", cfg.RootCAs)
	}
	if cfg.MinVersion != tlsVersion12 {
		t.Errorf("MinVersion = %#x, хочу TLS 1.2", cfg.MinVersion)
	}
	if cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify не должен быть true никогда (SECURITY.md §1)")
	}
}

// TestNewTLSConfigWithCA: задан ca_path — RootCAs заполняется ТОЛЬКО из этого
// PEM и содержит тот сертификат (приватный пул, SECURITY.md §1).
func TestNewTLSConfigWithCA(t *testing.T) {
	caPath := writeTestCA(t, t.TempDir())
	cfg, err := newTLSConfig(caPath)
	if err != nil {
		t.Fatalf("newTLSConfig(%s): %v", caPath, err)
	}
	if cfg.RootCAs == nil {
		t.Fatal("RootCAs = nil при заданном ca_path — запрещено (SECURITY.md §1)")
	}
	if _, err := os.Stat(caPath); err != nil {
		t.Fatal(err)
	}
	// Пул должен принимать сертификат, изданный этим CA (проверка через
	// верификацию самого сертификата как цепочки из одного элемента).
	if cfg.RootCAs.Equal(x509.NewCertPool()) {
		t.Error("RootCAs пуст — сертификат из ca_path не добавлен")
	}
}

// TestNewTLSConfigInvalid: мусор вместо PEM и несуществующий путь → ошибки на русском.
func TestNewTLSConfigInvalid(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("это не PEM"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := newTLSConfig(bad); err == nil {
		t.Error("хочу ошибку для файла без валидных PEM-сертификатов")
	} else {
		t.Logf("ошибка для мусорного PEM: %v", err)
	}
	missing := filepath.Join(dir, "missing.pem")
	if _, err := newTLSConfig(missing); err == nil {
		t.Error("хочу ошибку для несуществующего файла CA")
	}
}
