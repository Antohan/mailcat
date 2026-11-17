package ews

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Azure/go-ntlmssp"
)

// Credentials — учётные данные EWS для Basic/NTLM. Password не включается
// в строковые представления и никогда не попадает в логи и ошибки
// (docs/SECURITY.md §2).
type Credentials struct {
	Username string // DOMAIN\user или user@company.example
	Password string
}

// String маскирует креденшелы в fmt-выводе: только имя пользователя.
func (c Credentials) String() string { return c.Username }

// authTransport — RoundTripper со схемой «Basic, затем NTLM»
// (docs/EWS-REFERENCE.md §1–2): первый запрос уходит с Basic-заголовком;
// если сервер отвечает 401 с предложением NTLM/Negotiate, запрос повторяется
// через ntlmssp.Negotiator (креденшелы Negotiator берёт из того же
// Basic-заголовка запроса).
type authTransport struct {
	base  http.RoundTripper // *http.Transport с изолированным TLS-пулом
	ntlm  http.RoundTripper // ntlmssp.Negotiator поверх base
	creds Credentials

	basicOnce sync.Once // выбор схемы логируем один раз на сессию
	ntlmOnce  sync.Once
	log       *slog.Logger
}

// newHTTPClient собирает http.Client на сессию:
//   - TLS-конфиг из ca_path (приватный пул — transport.go, SECURITY.md §1);
//   - Basic → NTLM fallback (EWS-REFERENCE.md §2);
//   - редиректы на другой хост запрещены (SECURITY.md §7).
//
// Таймауты задаются не здесь, а контекстом в Client.Do: общий 30с,
// GetAttachment — 5 минут (SECURITY.md §7).
func newHTTPClient(tlsCfg *tls.Config, creds Credentials, log *slog.Logger) *http.Client {
	if log == nil {
		log = discardLogger()
	}
	base := &http.Transport{TLSClientConfig: tlsCfg}
	return &http.Client{
		Transport: &authTransport{
			base:  base,
			ntlm:  ntlmssp.Negotiator{RoundTripper: base},
			creds: creds,
			log:   log,
		},
		CheckRedirect: checkRedirectNoHostChange,
	}
}

// checkRedirectNoHostChange разрешает следовать редиректам только в пределах
// того же хоста (порт учитывается) и только по https: даунгрейд https→http
// на том же хосте утечёт Basic-креденшелами в открытом виде — запрещён
// (SECURITY.md §7, security-ревью этапа 1).
func checkRedirectNoHostChange(req *http.Request, via []*http.Request) error {
	if len(via) > 0 && req.URL.Host != via[len(via)-1].URL.Host {
		return fmt.Errorf("редирект на другой хост запрещён: %s → %s",
			via[len(via)-1].URL.Host, req.URL.Host)
	}
	if req.URL.Scheme != "https" {
		return fmt.Errorf("редирект с понижением безопасности запрещён (https → %s), хост %s",
			req.URL.Scheme, req.URL.Host)
	}
	return nil
}

// RoundTrip выполняет попытку Basic, при 401 с NTLM/Negotiate повторяет запрос
// через NTLM-транспорт. Изменять входящий запрос нельзя (контракт RoundTripper),
// поэтому для каждой попытки клонируется копия с собственным телом.
func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("чтение тела запроса: %w", err)
		}
	}

	// Попытка 1: Basic.
	basicReq, err := cloneWithBody(req, body)
	if err != nil {
		return nil, err
	}
	basicReq.SetBasicAuth(t.creds.Username, t.creds.Password)
	resp, err := t.base.RoundTrip(basicReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusUnauthorized || !offersNTLM(resp.Header.Values("WWW-Authenticate")) {
		t.basicOnce.Do(func() {
			t.log.Info("EWS-аутентификация: схема basic принята сервером", "scheme", "basic") // без учётных данных — SECURITY.md §2
		})
		return resp, nil // включая финальный 401 без NTLM — Do превратит его в ошибку аутентификации
	}
	t.ntlmOnce.Do(func() {
		t.log.Info("EWS-аутентификация: сервер отклонил Basic и предложил NTLM, переключаюсь на NTLM",
			"scheme", "ntlm") // без учётных данных — SECURITY.md §2
	})
	drainAndClose(resp)

	// Попытка 2: NTLM (Negotiator сам разыграет handshake; тело повторно
	// устанавливается им из Basic-заголовка запроса).
	ntlmReq, err := cloneWithBody(req, body)
	if err != nil {
		return nil, err
	}
	ntlmReq.SetBasicAuth(t.creds.Username, t.creds.Password)
	return t.ntlm.RoundTrip(ntlmReq)
}

// cloneWithBody клонирует запрос и ставит тело из буфера (клонирование
// обязательно: заголовок Authorization менять нельзя на исходном запросе).
func cloneWithBody(req *http.Request, body []byte) (*http.Request, error) {
	cl := req.Clone(req.Context())
	cl.Header = req.Header.Clone()
	if body == nil {
		cl.Body = nil
		return cl, nil
	}
	cl.Body = io.NopCloser(bytes.NewReader(body))
	cl.ContentLength = int64(len(body))
	cl.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	return cl, nil
}

// offersNTLM сообщает, предлагает ли сервер NTLM или Negotiate в 401-ответе.
func offersNTLM(challenges []string) bool {
	for _, ch := range challenges {
		ch = normalizeChallenge(ch)
		if ch == "NTLM" || ch == "Negotiate" {
			return true
		}
	}
	return false
}

// normalizeChallenge оставляет имя схемы (то, что до пробела) в верхнем регистре.
func normalizeChallenge(ch string) string {
	for i := 0; i < len(ch); i++ {
		if ch[i] == ' ' || ch[i] == '=' {
			return toUpper(ch[:i])
		}
	}
	return toUpper(ch)
}

func toUpper(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

// drainAndClose освобождает тело ответа, чтобы соединение переиспользовалось.
func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
}

var (
	discardOnce sync.Once
	discardSlog *slog.Logger
)

// discardLogger — логгер «в никуда», чтобы не проверять nil на каждом шаге.
func discardLogger() *slog.Logger {
	discardOnce.Do(func() {
		discardSlog = slog.New(slog.NewTextHandler(io.Discard, nil))
	})
	return discardSlog
}
