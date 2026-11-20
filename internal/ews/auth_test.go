package ews

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient собирает http.Client, доверяющий сертификату тестового сервера
// через приватный пул (механика SECURITY.md §1, TESTING.md §3).
func newTestClient(t *testing.T, srv *httptest.Server, creds Credentials, log *slog.Logger) *http.Client {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	return newHTTPClient(&tls.Config{RootCAs: pool, MinVersion: tlsVersion12}, creds, log)
}

// recordingHandler считает запросы и сохраняет заголовок Authorization.
type recordingHandler struct {
	count    atomic.Int64
	authSeen []string // добавляется из обработчика одного теста (без гонок)
}

func (h *recordingHandler) record(r *http.Request) {
	h.count.Add(1)
	h.authSeen = append(h.authSeen, r.Header.Get("Authorization"))
}

// TestAuthBasicAccepted: сервер принимает Basic — клиент проходит без переключений.
func TestAuthBasicAccepted(t *testing.T) {
	var rec recordingHandler
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		user, pass, ok := r.BasicAuth()
		if !ok || user != "ivanov" || pass != "secret-пароль" {
			w.Header().Set("WWW-Authenticate", `Basic realm="ews"`)
			http.Error(w, "нет Basic", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "ivanov", Password: "secret-пароль"}, nil)
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("статус = %d, хочу 200", resp.StatusCode)
	}
	if got := rec.count.Load(); got != 1 {
		t.Errorf("запросов %d, хочу 1 (без повторов при успешном Basic)", got)
	}
}

// TestAuthDeniedNoNTLM: 401 без предложения NTLM/Negotiate — клиент не ретраит,
// наружу отдаётся ответ 401 (верхний слой превратит его в ошибку аутентификации).
func TestAuthDeniedNoNTLM(t *testing.T) {
	var rec recordingHandler
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		w.Header().Set("WWW-Authenticate", `Basic realm="ews"`)
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "ivanov", Password: "wrong"}, nil)
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("статус = %d, хочу 401", resp.StatusCode)
	}
	if got := rec.count.Load(); got != 1 {
		t.Errorf("запросов %d, хочу 1 (нет NTLM — нет и повтора)", got)
	}
}

// TestAuthNTLMSwitch: 401 с WWW-Authenticate: NTLM — клиент переключается на
// NTLM-транспорт (виден Authorization: NTLM ...) и не падает; handshake не мокаем,
// итоговый ответ — 401.
func TestAuthNTLMSwitch(t *testing.T) {
	var rec recordingHandler
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.record(r)
		w.Header().Set("WWW-Authenticate", "NTLM")
		http.Error(w, "challenge", http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "COMPANY\\ivanov", Password: "secret"}, nil)
	resp, err := client.Post(srv.URL, "text/xml; charset=utf-8", bytes.NewReader([]byte("<x/>")))
	if err != nil {
		t.Fatalf("POST: %v (клиент не должен падать на NTLM-челлендже)", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("статус = %d, хочу 401 (handshake в моке не завершается)", resp.StatusCode)
	}
	if got := rec.count.Load(); got < 2 {
		t.Errorf("запросов %d, хочу как минимум 2 — клиент должен переключиться на NTLM", got)
	}
	switched := false
	for _, a := range rec.authSeen {
		if len(a) > 4 && a[:4] == "NTLM" {
			switched = true
		}
	}
	if !switched {
		t.Errorf("ни один запрос не ушёл с Authorization: NTLM — переключения не было; заголовки: %v", rec.authSeen)
	}
}

// TestAuthNTLMBodyReplay: при переключении на NTLM тело POST-запроса доходит целиком
// (переключение повторяет запрос с тем же конвертом).
func TestAuthNTLMBodyReplay(t *testing.T) {
	body := []byte("<soap:Envelope>тело-конверта-кириллица</soap:Envelope>")
	var lastBody string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "" {
			lastBody = string(b)
		}
		w.Header().Set("WWW-Authenticate", "NTLM")
		http.Error(w, "challenge", http.StatusUnauthorized)
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "ivanov@company.example", Password: "secret"}, nil)
	resp, err := client.Post(srv.URL, "text/xml; charset=utf-8", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if lastBody != string(body) {
		t.Errorf("тело при NTLM-попытке = %q, хочу исходное %q", lastBody, string(body))
	}
}

// TestAuthSchemeLogged: факт выбора схемы попадает в лог, пароль — нет (SECURITY.md §2).
func TestAuthSchemeLogged(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))

	basicSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); !ok {
			http.Error(w, "denied", http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, "ok")
	}))
	defer basicSrv.Close()
	if _, err := newTestClient(t, basicSrv, Credentials{Username: "ivanov", Password: "parol"}, log).Get(basicSrv.URL); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buf.Bytes(), []byte("parol")) {
		t.Errorf("в лог попал пароль: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("basic")) {
		t.Errorf("в логе нет упоминания схемы basic: %s", buf.String())
	}

	ntlmSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "NTLM")
		http.Error(w, "challenge", http.StatusUnauthorized)
	}))
	defer ntlmSrv.Close()
	buf.Reset()
	resp, err := newTestClient(t, ntlmSrv, Credentials{Username: "ivanov", Password: "parol"}, log).Get(ntlmSrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if bytes.Contains(buf.Bytes(), []byte("parol")) {
		t.Errorf("в лог попал пароль: %s", buf.String())
	}
	if !bytes.Contains(buf.Bytes(), []byte("NTLM")) {
		t.Errorf("в логе нет переключения на NTLM: %s", buf.String())
	}
}

// TestCredentialsString: строковое представление креденшелов не выдаёт пароль.
func TestCredentialsString(t *testing.T) {
	c := Credentials{Username: "ivanov", Password: "parol"}
	if s := fmt.Sprintf("%v %+v", c, c); bytes.Contains([]byte(s), []byte("parol")) {
		t.Errorf("пароль утёк через fmt: %s", s)
	}
}

// TestRedirectHostChangeRejected: редирект на другой хост отклоняется (SECURITY.md §7).
func TestRedirectHostChangeRejected(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://evil.example/EWS/Exchange.asmx", http.StatusFound)
	}))
	defer srv.Close()

	client := newTestClient(t, srv, Credentials{Username: "u", Password: "p"}, nil)
	_, err := client.Get(srv.URL)
	if err == nil {
		t.Fatal("редирект на другой хост должен быть отклонён")
	}
	t.Logf("ошибка редиректа: %v", err)
}

// TestRequestContextTimeout: отменённый контекст прерывает запрос.
func TestRequestContextTimeout(t *testing.T) {
	done := make(chan struct{})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-done
	}))
	defer srv.Close()
	defer close(done)

	client := newTestClient(t, srv, Credentials{Username: "u", Password: "p"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Do(req)
	if err == nil {
		t.Fatal("хочу ошибку по таймауту контекста")
	}
}
