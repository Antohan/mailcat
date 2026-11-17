package ews

// ewsmock_test.go — интеграционный mock-сервер EWS (docs/TESTING.md §3):
// один TLS-сервер раздаёт фикстуры по содержимому запроса; FindItem учитывает
// Offset (2 страницы по 3 письма — сквозная пагинация); режим denied всегда
// отвечает 401. Клиент ходит через приватный пул сертификата сервера — заодно
// проверяется механика SECURITY.md §1.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// ewsMock — тестовый «сервер Exchange».
type ewsMock struct {
	*httptest.Server
	requests atomic.Int64
}

// newEWSMock поднимает сервер в режиме basic (все операции успешны).
func newEWSMock(t *testing.T) *ewsMock {
	t.Helper()
	return newEWSMockMode(t, "basic")
}

// newEWSMockMode: basic — фикстуры по операциям; denied — всегда 401.
func newEWSMockMode(t *testing.T, mode string) *ewsMock {
	t.Helper()
	m := &ewsMock{}
	m.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.requests.Add(1)
		if mode == "denied" {
			w.Header().Set("WWW-Authenticate", `Basic realm="ews"`)
			http.Error(w, "denied", http.StatusUnauthorized)
			return
		}
		body, _ := io.ReadAll(r.Body)
		serveFixture(t, w, string(body))
	}))
	t.Cleanup(m.Close)
	return m
}

// serveFixture выбирает фикстуру по содержимому SOAP-запроса. CalendarView
// проверяется раньше FindItem: встречный запрос тоже содержит элемент FindItem.
func serveFixture(t *testing.T, w http.ResponseWriter, reqBody string) {
	t.Helper()
	switch {
	case strings.Contains(reqBody, "FindFolder"):
		_, _ = w.Write(fixture(t, "findfolder_response.xml"))
	case strings.Contains(reqBody, "GetItem"):
		_, _ = w.Write(fixture(t, "getitem_response.xml"))
	case strings.Contains(reqBody, "GetAttachment"):
		_, _ = w.Write(fixture(t, "getattachment_response.xml"))
	case strings.Contains(reqBody, "CalendarView"):
		_, _ = w.Write(fixture(t, "calendar_response.xml"))
	case strings.Contains(reqBody, "FindItem"):
		// Пагинация по Offset: первая страница при Offset="0" (или без вида),
		// вторая — при любом другом Offset.
		if strings.Contains(reqBody, `Offset="0"`) || !strings.Contains(reqBody, "Offset=") {
			_, _ = w.Write(fixture(t, "finditem_response_p1.xml"))
		} else {
			_, _ = w.Write(fixture(t, "finditem_response_p2.xml"))
		}
	default:
		http.Error(w, "неизвестная операция", http.StatusBadRequest)
	}
}

// TestMockSmokeAllOperations: все операции на одном сервере, сквозной сценарий
// «папки → список → письмо → вложение → календарь».
func TestMockSmokeAllOperations(t *testing.T) {
	mock := newEWSMock(t)
	client := newTestEWSClient(t, mock.Server)
	ctx := context.Background()

	folders, err := client.FindFolders(ctx)
	if err != nil {
		t.Fatalf("FindFolders: %v", err)
	}
	if len(folders) != 3 || folders[0].DisplayName != "Входящие" || folders[0].UnreadCount != 23 {
		t.Fatalf("папки = %+v", folders)
	}

	// Сквозная пагинация: страница 1 → NextOffset=3, страница 2 → последняя.
	page1, err := client.FindDistinguishedMessages(ctx, "inbox", 3, 0)
	if err != nil {
		t.Fatalf("FindMessages стр.1: %v", err)
	}
	if len(page1.Items) != 3 || page1.IncludesLast {
		t.Fatalf("стр.1 = %+v", page1)
	}
	next, more := NextOffset(page1)
	if !more || next != 3 {
		t.Fatalf("NextOffset(стр.1) = %d,%v — хочу 3,true", next, more)
	}
	page2, err := client.FindDistinguishedMessages(ctx, "inbox", 3, next)
	if err != nil {
		t.Fatalf("FindMessages стр.2: %v", err)
	}
	if !page2.IncludesLast {
		t.Fatalf("стр.2 не последняя: %+v", page2)
	}
	if _, more := NextOffset(page2); more {
		t.Fatal("после последней страницы листать нельзя")
	}

	// Письмо целиком: тело + 2 вложения.
	itemID := page1.Items[0].ItemID
	changeKey := page1.Items[0].ChangeKey
	detail, err := client.GetMessage(ctx, itemID, changeKey)
	if err != nil {
		t.Fatalf("GetMessage: %v", err)
	}
	if detail.BodyText == "" || len(detail.Attachments) != 2 {
		t.Fatalf("письмо = %+v", detail)
	}

	// Вложение: лимит больше размера — байты декодированы.
	data, err := client.GetAttachment(ctx, detail.Attachments[0], 10*1024*1024)
	if err != nil {
		t.Fatalf("GetAttachment: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("пустое вложение")
	}

	// Календарь: 3 встречи, Occurrence среди них.
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)
	events, err := client.FindEvents(ctx, start, end)
	if err != nil {
		t.Fatalf("FindEvents: %v", err)
	}
	if len(events) != 3 || events[0].ItemType != "Occurrence" {
		t.Fatalf("события = %+v", events)
	}
}

// TestMockDenied: сервер всегда 401 — клиент отдаёт *HTTPError со статусом 401
// (аутентификация не пройдена; NTLM-рукопожатие в mock не разыгрывается).
func TestMockDenied(t *testing.T) {
	mock := newEWSMockMode(t, "denied")
	client := newTestEWSClient(t, mock.Server)
	_, err := client.FindFolders(context.Background())
	if err == nil {
		t.Fatal("хочу ошибку аутентификации")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		t.Fatalf("ошибка %v (%T), хочу *HTTPError", err, err)
	}
	if httpErr.Status != http.StatusUnauthorized {
		t.Errorf("Status = %d, хочу 401", httpErr.Status)
	}
}
