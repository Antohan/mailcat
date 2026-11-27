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
	"log/slog"
	"net"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

// Таймауты запросов (docs/SECURITY.md §7).
const (
	requestTimeout    = 30 * time.Second // общий на запрос
	attachmentTimeout = 5 * time.Minute  // GetAttachment (большие вложения)
)

// requestVersion — версия сервера в SOAP-заголовке (docs/EWS-REFERENCE.md §3).
const requestVersion = "Exchange2016"

// maxResponseBytes — потолок размера SOAP-ответа: защищает от OOM при
// неадекватно большом конверте (security-ревью этапа 1, §5); обычные ответы
// (включая вложения до лимита конфигурации) значительно меньше.
var maxResponseBytes int64 = 512 << 20

// Client — клиент EWS на сессию: один http.Client (Basic→NTLM, изолированный
// TLS) и адрес endpoint https://<server>/EWS/Exchange.asmx.
type Client struct {
	HTTP *http.Client
	URL  string
}

// NewClient собирает Client: URL endpoint, креденшелы, путь к PEM корпоративного
// CA (пусто — системные корни) и логгер (nil — логи отбрасываются).
// Принимает только https-адрес (SECURITY.md §7).
func NewClient(url string, creds Credentials, caPath string, log *slog.Logger) (*Client, error) {
	if u, err := neturl.Parse(url); err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, fmt.Errorf("адрес EWS должен быть https, получено %q", url)
	}
	tlsCfg, err := newTLSConfig(caPath)
	if err != nil {
		return nil, err
	}
	return &Client{HTTP: newHTTPClient(tlsCfg, creds, log), URL: url}, nil
}

// Do отправляет операцию EWS (request — типизированный элемент вида m:FindItem)
// внутри SOAP-конверта и разбирает ответ в response (указатель на структуру
// вида m:FindItemResponse). HTTP-статусы, soap:Fault и ошибки конверта
// превращаются в типизированные ошибки: *HTTPError, *SoapFaultError, *ResponseError
// (последний ставят сами операции по ResponseClass).
func (c *Client) Do(ctx context.Context, request any, response any) error {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	}

	env := &requestEnvelope{}
	env.Header.RequestServerVersion.Version = requestVersion
	env.Body.Operation = request
	payload, err := xml.Marshal(env)
	if err != nil {
		return fmt.Errorf("сборка SOAP-конверта: %w", err)
	}
	body := append([]byte(xml.Header), payload...)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("создание HTTP-запроса EWS: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml; charset=utf-8") // EWS-REFERENCE.md §1

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return transportError(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		// WHY: Exchange кладёт soap:Fault прямо в тело не-200 ответа
		// Why: живой сервер отвечал HTTP 500 + ErrorInvalidRequest, а голый
		//  статус скрывал причину; код из fault важнее.
		// Not to do: выбрасывать тело без разбора (было — «неожиданный статус»).
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var env responseEnvelope
		if err := xml.Unmarshal(data, &env); err == nil && env.Body.Fault != nil {
			code := env.Body.Fault.FaultCode
			if i := strings.Index(code, ":"); i >= 0 {
				code = code[i+1:] // «a:ErrorInvalidRequest» → «ErrorInvalidRequest»
			}
			return &SoapFaultError{Code: code, String: env.Body.Fault.FaultString}
		}
		return &HTTPError{Status: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("чтение ответа EWS: %w", err)
	}
	if int64(len(data)) > maxResponseBytes {
		return fmt.Errorf("размер ответа EWS превышает потолок %d байт — соединение прервано", maxResponseBytes)
	}
	return parseResponse(data, response)
}

// transportError переводит транспортные сбои на русский язык: проблемы
// сертификата — с объяснением и без обхода (SECURITY.md §1), обрывы сети —
// с подсказкой про VPN (AGENTS.md), таймауты — явно.
func transportError(err error) error {
	var certErr *tls.CertificateVerificationError
	if errors.As(err, &certErr) {
		return fmt.Errorf("сервер представил сертификат, который не подписан указанным в ca_path корнем (обход проверки невозможен): %w", err)
	}
	var unknownCA x509.UnknownAuthorityError
	if errors.As(err, &unknownCA) {
		return fmt.Errorf("сертификат сервера не подписан известным корнем (проверьте ca_path в конфигурации): %w", err)
	}
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("запрос EWS отменён: %w", err)
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("превышен таймаут запроса EWS: %w", err)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Errorf("нет соединения с сервером Exchange (похоже, VPN выключен): %w", err)
	}
	return fmt.Errorf("запрос EWS: %w", err)
}

// requestEnvelope — каркас запроса строго по docs/EWS-REFERENCE.md §3.
// Namespace-URL в тегах — литералы (структура тега Go не допускает выражений);
// соответствуют константам ns* ниже.
type requestEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  requestHeader
	Body    requestBody
}

type requestHeader struct {
	XMLName              xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`
	RequestServerVersion struct {
		XMLName xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/types RequestServerVersion"`
		Version string   `xml:"Version,attr"`
	}
}

type requestBody struct {
	XMLName   xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Operation any      // элемент операции; имя берётся из XMLName значения
}

// responseEnvelope — минимальный каркас разбора ответа: нужен soap:Fault и
// «сырое» содержимое soap:Body (разбирается повторно в ответ операции,
// потому что реальные серверы объявляют пространства имён на самом ответе).
type responseEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Body    responseBody
}

type responseBody struct {
	XMLName xml.Name   `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`
	Fault   *soapFault `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	Inner   []byte     `xml:",innerxml"`
}

type soapFault struct {
	XMLName     xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`
	FaultCode   string   `xml:"faultcode"`
	FaultString string   `xml:"faultstring"`
}

// parseResponse разбирает полный SOAP-ответ: soap:Fault → *SoapFaultError,
// иначе содержимое soap:Body — в response. Теги ответа операции сопоставляются
// по локальным именам (без префиксов) — конверт реального сервера может
// использовать любые префиксы.
func parseResponse(data []byte, response any) error {
	var env responseEnvelope
	if err := xml.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("разбор SOAP-конверта ответа: %w", err)
	}
	if env.Body.Fault != nil {
		return &SoapFaultError{Code: env.Body.Fault.FaultCode, String: env.Body.Fault.FaultString}
	}
	if response == nil {
		return nil
	}
	if len(bytes.TrimSpace(env.Body.Inner)) == 0 {
		return errors.New("пустое тело SOAP-ответа EWS")
	}
	if err := xml.Unmarshal(env.Body.Inner, response); err != nil {
		return fmt.Errorf("разбор ответа операции EWS: %w", err)
	}
	return nil
}

// HTTPError — EWS ответил неожиданным HTTP-статусом (EWS-REFERENCE.md §9).
type HTTPError struct {
	Status int
}

func (e *HTTPError) Error() string {
	switch e.Status {
	case http.StatusUnauthorized:
		return "ошибка аутентификации (HTTP 401): проверьте имя пользователя и пароль"
	case http.StatusForbidden:
		return "доступ запрещён (HTTP 403): возможно, EWS отключён политикой для вашего ящика — спросите ИТ-отдел"
	case http.StatusServiceUnavailable:
		return "служба EWS недоступна (HTTP 503): попробуйте позже"
	default:
		return fmt.Sprintf("неожиданный HTTP-статус от EWS: %d", e.Status)
	}
}

// SoapFaultError — ошибка конверта soap:Fault (EWS-REFERENCE.md §9).
type SoapFaultError struct {
	Code   string // faultcode
	String string // faultstring
}

func (e *SoapFaultError) Error() string {
	return fmt.Sprintf("SOAP-fault %s: %s", e.Code, e.String)
}

// ResponseError — ResponseClass="Error" из m:*ResponseMessage (EWS-REFERENCE.md §9).
type ResponseError struct {
	Code    string // ResponseCode, например ErrorAccessDenied
	Message string // MessageText
}

func (e *ResponseError) Error() string {
	if e.Message == "" {
		return "EWS: " + e.Code
	}
	return fmt.Sprintf("EWS %s: %s", e.Code, e.Message)
}

// UserHint возвращает человеко-понятную подсказку для известных кодов EWS.
func (e *ResponseError) UserHint() string {
	switch e.Code {
	case "ErrorAccessDenied":
		return "нет прав на операцию — спросите ИТ-отдел"
	case "ErrorItemNotFound":
		return "письмо удалено или перемещено — обновите список"
	case "ErrorFolderNotFound":
		return "папка не найдена — обновите список папок"
	case "ErrorConnectionFailed":
		return "нет связи с Exchange — похоже, VPN выключен"
	case "ErrorInternalServerError":
		return "временная ошибка на стороне сервера — попробуйте позже"
	default:
		return ""
	}
}

// checkResponseClass превращает ResponseClass="Error" ответа операции
// в *ResponseError; Success и Warning ошибкой не считаются.
func checkResponseClass(class, code, messageText string) error {
	if class != "Error" {
		return nil
	}
	return &ResponseError{Code: code, Message: messageText}
}
