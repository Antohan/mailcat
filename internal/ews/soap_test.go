package ews

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fixture читает тестовый XML из testdata.
func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("чтение фикстуры %s: %v", name, err)
	}
	return data
}

// TestParseResponseFault: SOAP-fault из фикстуры → типизированная ошибка
// с faultcode и faultstring (EWS-REFERENCE.md §9).
func TestParseResponseFault(t *testing.T) {
	var resp struct{} // fault должен вернуть ошибку до разбора ответа операции
	err := parseResponse(fixture(t, "soap_fault.xml"), &resp)
	if err == nil {
		t.Fatal("хочу SoapFaultError")
	}
	var fault *SoapFaultError
	if !errors.As(err, &fault) {
		t.Fatalf("ошибка %v не SoapFaultError", err)
	}
	if fault.Code != "soap:Client" {
		t.Errorf("Code = %q, хочу soap:Client", fault.Code)
	}
	if fault.String != "Запрос содержит недопустимые значения" {
		t.Errorf("String = %q", fault.String)
	}
	if msg := fault.Error(); !bytes.Contains([]byte(msg), []byte("SOAP-fault")) {
		t.Errorf("Error() = %q, хочу упоминание SOAP-fault", msg)
	}
}

// TestParseResponseOK: без fault — тело soap:Body разбирается в ответ операции.
func TestParseResponseOK(t *testing.T) {
	doc := []byte(`<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"` +
		` xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">` +
		`<soap:Body><m:PingResponse><m:Value>привет</m:Value></m:PingResponse></soap:Body></soap:Envelope>`)
	var resp struct {
		XMLName xml.Name `xml:"PingResponse"` // локальный тестовый ответ, не EWS-шаблон
		Value   string   `xml:"Value"`
	}
	if err := parseResponse(doc, &resp); err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if resp.Value != "привет" {
		t.Errorf("Value = %q, хочу «привет»", resp.Value)
	}
}

// testOperation — минимальная операция для проверки транспорта Client.Do
// (не EWS-шаблон; реальные конверты покрывают golden-тесты операций).
type testOperation struct {
	XMLName xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/messages Ping"`
	Query   string   `xml:"query,attr"`
}

// newTestEWSClient — Client на тестовом TLS-сервере с приватным пулом сервера.
func newTestEWSClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return &Client{
		HTTP: newHTTPClient(&tls.Config{RootCAs: pool, MinVersion: tlsVersion12}, Credentials{Username: "u", Password: "p"}, nil),
		URL:  srv.URL,
	}
}

// TestDoHTTPStatusErrors: HTTP 401/403/503/500 → типизированная HTTPError
// с сообщениями на русском (EWS-REFERENCE.md §9).
func TestDoHTTPStatusErrors(t *testing.T) {
	cases := []struct {
		status      int
		wantMessage string
	}{
		{401, "аутентификац"},
		{403, "ИТ-отдел"},
		{503, "недоступна"},
		{500, "500"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.status), func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "отказ", tc.status)
			}))
			defer srv.Close()
			client := newTestEWSClient(t, srv)
			err := client.Do(context.Background(), &testOperation{Query: "q"}, nil)
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("ошибка %v (%T), хочу *HTTPError", err, err)
			}
			if httpErr.Status != tc.status {
				t.Errorf("Status = %d, хочу %d", httpErr.Status, tc.status)
			}
			if !bytes.Contains([]byte(httpErr.Error()), []byte(tc.wantMessage)) {
				t.Errorf("Error() = %q, хочу упоминание %q", httpErr.Error(), tc.wantMessage)
			}
		})
	}
}

// TestDoFaultOnHTTPError: Exchange кладёт soap:Fault прямо в тело не-200
// ответа (наблюдалось на живом сервере: HTTP 500 + ErrorInvalidRequest) —
// код ошибки из fault важнее голого статуса, вытаскиваем его.
func TestDoFaultOnHTTPError(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/"><s:Body><s:Fault><faultcode xmlns:a="http://schemas.microsoft.com/exchange/services/2006/types">a:ErrorInvalidRequest</faultcode><faultstring xml:lang="ru-RU">The request is invalid.</faultstring></s:Fault></s:Body></s:Envelope>`))
	}))
	defer srv.Close()
	client := newTestEWSClient(t, srv)
	err := client.Do(context.Background(), &testOperation{Query: "q"}, nil)
	var fault *SoapFaultError
	if !errors.As(err, &fault) {
		t.Fatalf("ошибка %v (%T), хочу *SoapFaultError", err, err)
	}
	if fault.Code != "ErrorInvalidRequest" {
		t.Errorf("Code = %q, хочу ErrorInvalidRequest (без префикса a:)", fault.Code)
	}
	if !bytes.Contains([]byte(fault.Error()), []byte("The request is invalid")) {
		t.Errorf("Error() = %q, без текста fault", fault.Error())
	}
}

// TestDoSuccess: 200 с корректным конвертом — ответ разбирается, Content-Type
// запроса text/xml; charset=utf-8.
func TestDoSuccess(t *testing.T) {
	var gotCT string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("<Ping")) {
			t.Errorf("в теле запроса нет операции Ping: %s", body)
		}
		gotCT = r.Header.Get("Content-Type")
		_, _ = fmt.Fprint(w, `<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"`+
			` xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">`+
			`<soap:Body><m:PingResponse><m:Value>ок</m:Value></m:PingResponse></soap:Body></soap:Envelope>`)
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	var resp struct {
		XMLName xml.Name `xml:"PingResponse"`
		Value   string   `xml:"Value"`
	}
	if err := client.Do(context.Background(), &testOperation{Query: "q"}, &resp); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotCT != "text/xml; charset=utf-8" {
		t.Errorf("Content-Type = %q, хочу text/xml; charset=utf-8", gotCT)
	}
	if resp.Value != "ок" {
		t.Errorf("Value = %q, хочу «ок»", resp.Value)
	}
}

// TestDoTLSErrorExplanation: сертификат сервера не в приватном пуле — ошибка
// на русском с объяснением и без предложения обхода (SECURITY.md §1).
func TestDoTLSErrorExplanation(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()

	// Пул из «чужого» CA, не подписавшего сертификат тестового сервера.
	otherCA := writeTestCA(t, t.TempDir())
	tlsCfg, err := newTLSConfig(otherCA)
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{
		HTTP: newHTTPClient(tlsCfg, Credentials{Username: "u", Password: "p"}, nil),
		URL:  srv.URL,
	}
	err = client.Do(context.Background(), &testOperation{}, nil)
	if err == nil {
		t.Fatal("хочу ошибку проверки сертификата")
	}
	msg := err.Error()
	for _, want := range []string{"сертификат", "ca_path"} {
		if !bytes.Contains([]byte(msg), []byte(want)) {
			t.Errorf("ошибка %q не объясняет проблему (%q)", msg, want)
		}
	}
	if bytes.Contains([]byte(msg), []byte("InsecureSkipVerify")) {
		t.Errorf("ошибка предлагает обход проверки: %q", msg)
	}
}

// TestDoNetworkErrorVPNHint: сервер недоступен — подсказка про VPN
// (AGENTS.md: Exchange виден только через VPN).
func TestDoNetworkErrorVPNHint(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close() // порт больше не слушается

	client := &Client{
		HTTP: newHTTPClient(&tls.Config{MinVersion: tlsVersion12}, Credentials{Username: "u", Password: "p"}, nil),
		URL:  url,
	}
	err := client.Do(context.Background(), &testOperation{}, nil)
	if err == nil {
		t.Fatal("хочу ошибку соединения")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("VPN")) {
		t.Errorf("ошибка %q без подсказки про VPN", err)
	}
}

// TestDoDeadlineExceeded: дедлайн контекста истекает — понятная ошибка таймаута.
func TestDoDeadlineExceeded(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-release
	}))
	defer srv.Close()
	defer close(release)

	client := newTestEWSClient(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := client.Do(ctx, &testOperation{}, nil)
	if err == nil {
		t.Fatal("хочу ошибку таймаута")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("таймаут")) {
		t.Errorf("ошибка %q не про таймаут", err)
	}
}

// TestEWSErrorUserHint: подсказки по кодам EWS (EWS-REFERENCE.md §9).
func TestEWSErrorUserHint(t *testing.T) {
	cases := map[string]string{
		"ErrorAccessDenied":        "ИТ-отдел",
		"ErrorItemNotFound":        "удалено",
		"ErrorFolderNotFound":      "папка не найдена",
		"ErrorConnectionFailed":    "VPN",
		"ErrorInternalServerError": "сервера",
		"NoError":                  "",
	}
	for code, want := range cases {
		e := &ResponseError{Code: code, Message: "текст"}
		hint := e.UserHint()
		if want == "" {
			if hint != "" {
				t.Errorf("%s: подсказка %q, хочу пустую", code, hint)
			}
			continue
		}
		if !bytes.Contains([]byte(hint), []byte(want)) {
			t.Errorf("%s: подсказка %q без %q", code, hint, want)
		}
		if msg := e.Error(); !bytes.Contains([]byte(msg), []byte(code)) {
			t.Errorf("Error() = %q без кода %s", msg, code)
		}
	}
}
