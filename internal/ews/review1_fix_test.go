package ews

// Тесты фиксов по security-ревью этапа 1:
// 1) запрет scheme-даунгрейда при редиректе и не-https URL в NewClient (§7);
// 2) лимит вложения при неизвестном ContentLength + потолок размера ответа (§5).

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestNewClientRejectsNonHTTPS: NewClient отклоняет не-https endpoint (§7).
func TestNewClientRejectsNonHTTPS(t *testing.T) {
	for _, url := range []string{"http://mail.example/EWS/Exchange.asmx", "ftp://mail.example"} {
		if _, err := NewClient(url, Credentials{Username: "u", Password: "p"}, "", nil); err == nil {
			t.Errorf("NewClient(%q) не отклонил не-https адрес", url)
		} else if !strings.Contains(err.Error(), "https") {
			t.Errorf("NewClient(%q): ошибка без упоминания https: %v", url, err)
		}
	}
}

// TestRedirectSchemeDowngradeRejected: редирект https→http на ТОМ ЖЕ хосте
// отклоняется — иначе Go переиспользует заголовок Authorization и пароль
// уйдёт в открытом виде (§7).
func TestRedirectSchemeDowngradeRejected(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// даунгрейд схемы на том же хосте (порт сохраняем)
		http.Redirect(w, r, "http://"+r.Host+"/EWS/Exchange.asmx", http.StatusFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "u", Password: "p"}, nil)
	resp, err := client.Get(srv.URL)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatal("редирект https→http должен быть отклонён")
	}
	if !strings.Contains(err.Error(), "запрещён") && !strings.Contains(err.Error(), "https") {
		t.Errorf("ошибка без объяснения про запрещённый даунгрейд: %v", err)
	}
}

// TestGetAttachmentUnknownSizeOverLimit: ContentLength неизвестен (0), но
// скачанное содержимое превышает лимит — отказ ПОСЛЕ декодирования, до
// выдачи данных наружу (§5).
func TestGetAttachmentUnknownSizeOverLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(fixture(t, "getattachment_response.xml"))
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	att := AttachmentMeta{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf"} // размер неизвестен
	_, err := client.GetAttachment(context.Background(), att, 10)            // лимит 10 байт, контент 31 байт
	if err == nil {
		t.Fatal("хочу отказ: скачанное содержимое больше лимита")
	}
	if !strings.Contains(err.Error(), "лимит") {
		t.Errorf("ошибка без упоминания лимита: %v", err)
	}
}

// TestDoResponseSizeLimit: ответ больше потолка конверта — ошибка, а не OOM.
func TestDoResponseSizeLimit(t *testing.T) {
	old := maxResponseBytes
	maxResponseBytes = 128 // 128 байт потолок на время теста
	defer func() { maxResponseBytes = old }()

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.Copy(w, io.LimitReader(zeroReader{}, 4096)) // 4 КБ «конверта»
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	err := client.Do(context.Background(), &testOperation{}, nil)
	if err == nil {
		t.Fatal("хочу ошибку превышения размера ответа")
	}
	if !strings.Contains(err.Error(), "размер") {
		t.Errorf("ошибка без упоминания размера: %v", err)
	}
}

// TestGetAttachmentStillWorksAfterLimits: фикстура проходит при достаточном
// лимите и неизвестном размере (регресс после пост-проверки).
func TestGetAttachmentStillWorksAfterLimits(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = w.Write(fixture(t, "getattachment_response.xml"))
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	att := AttachmentMeta{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf"}
	data, err := client.GetAttachment(context.Background(), att, 1024*1024)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(data) != "PDF-BINARY-CONTENT-mailcat-test-123" {
		t.Errorf("контент = %q", data)
	}
	if calls.Load() == 0 {
		t.Error("сервер не вызван")
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}

var _ = errors.Is // чтобы импорт не дрожал при правках
