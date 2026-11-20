package ews

// Тесты GetAttachment (docs/EWS-REFERENCE.md §7): лимит проверяется ДО
// скачивания (SECURITY.md §5), содержимое — base64 t:Content.

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestGetAttachmentRequestGolden: маршалинг запроса соответствует шаблону §7:
// m:GetAttachment/m:AttachmentShape(t:BaseShape IdOnly)/m:AttachmentIds(t:AttachmentId).
func TestGetAttachmentRequestGolden(t *testing.T) {
	data, err := xml.MarshalIndent(newGetAttachmentRequest("AAMkAGI2ATT1="), "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса GetAttachment: %v", err)
	}
	want := []byte(`<m:GetAttachment
	    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
	    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
	  <m:AttachmentShape><t:BaseShape>IdOnly</t:BaseShape></m:AttachmentShape>
	  <m:AttachmentIds><t:AttachmentId Id="AAMkAGI2ATT1="/></m:AttachmentIds>
	</m:GetAttachment>`)
	assertXMLLike(t, "запрос GetAttachment", data, want)
}

// TestGetAttachmentLimitExceeded: ContentLength больше лимита — отказ ДО запроса
// (сервер не дёргается вообще).
func TestGetAttachmentLimitExceeded(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	att := AttachmentMeta{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf", ContentType: "application/pdf", ContentLength: 2097152}
	_, err := client.GetAttachment(context.Background(), att, 1024)
	if err == nil {
		t.Fatal("хочу отказ по лимиту, получил nil")
	}
	for _, want := range []string{"report.pdf", "2097152", "лимит"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("ошибка %q без %q", err.Error(), want)
		}
	}
	if got := calls.Load(); got != 0 {
		t.Errorf("сервер вызван %d раз — лимит должен проверяться до запроса", got)
	}
}

// TestGetAttachmentOK: фикстура с base64-контентом декодируется в байты.
func TestGetAttachmentOK(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "GetAttachment") {
			t.Errorf("запрос не похож на GetAttachment: %s", body)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write(fixture(t, "getattachment_response.xml"))
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	att := AttachmentMeta{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf", ContentType: "application/pdf", ContentLength: 31}
	data, err := client.GetAttachment(context.Background(), att, 1024*1024)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if string(data) != "PDF-BINARY-CONTENT-mailcat-test-123" {
		t.Errorf("декодировано %q, хочу PDF-BINARY-CONTENT-mailcat-test-123", data)
	}
}

// TestGetAttachmentUnknownSize: ContentLength=0 (размер неизвестен) — запрос
// уходит, содержимое возвращается.
func TestGetAttachmentUnknownSize(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(fixture(t, "getattachment_response.xml"))
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	att := AttachmentMeta{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf"} // размер неизвестен
	data, err := client.GetAttachment(context.Background(), att, 1024*1024)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if len(data) == 0 {
		t.Error("пустое содержимое вложения")
	}
}
