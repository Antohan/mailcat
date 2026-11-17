// Package mail — доменные модели почты и календаря: Folder, MessageSummary,
// Message, Attachment, Event. Пакет не ходит в сеть (за этим следит тест-сторож
// в mail_test.go) и не знает про SOAP: конвертация из структур internal/ews —
// отдельными функциями в этом пакете.
package mail

import (
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// Address — адрес корреспондента: отображаемое имя + e-mail.
type Address struct {
	Name  string
	Email string
}

// ParseAddress разбирает строку вида "Имя <user@host>" или "user@host".
func ParseAddress(s string) (Address, error) {
	a, err := mail.ParseAddress(s)
	if err != nil {
		return Address{}, fmt.Errorf("разбор адреса %q: %w", s, err)
	}
	return Address{Name: a.Name, Email: a.Address}, nil
}

// String возвращает "Имя <email>" либо голый адрес.
func (a Address) String() string {
	if a.Name == "" {
		return a.Email
	}
	return a.Name + " <" + a.Email + ">"
}

// Folder — почтовая папка из FindFolder.
type Folder struct {
	ID               string
	ChangeKey        string
	DisplayName      string
	TotalCount       int // всего писем
	UnreadCount      int // непрочитанных
	ChildFolderCount int // вложенных папок
}

// MessageSummary — строка списка писем (FindItem).
type MessageSummary struct {
	ItemID         string
	ChangeKey      string
	Subject        string
	Received       time.Time
	IsRead         bool
	HasAttachments bool
	From           Address
}

// Message — полное письмо (GetItem).
type Message struct {
	MessageSummary
	Sent              time.Time
	InternetMessageID string
	To                []Address
	Cc                []Address
	// BodyHTML — тело письма как вернул сервер (может быть пусто, если тело текстовое).
	BodyHTML string
	// BodyText — текстовое тело: TextBody сервера либо конвертация HTML (html2text).
	BodyText    string
	Attachments []Attachment
}

// Attachment — вложение; байты не хранятся, скачиваются отдельно (GetAttachment).
type Attachment struct {
	AttachmentID  string
	Name          string
	ContentType   string
	ContentLength int64
}

// SizeLabel возвращает человекочитаемый размер: «512 Б», «2.0 МБ».
func (a Attachment) SizeLabel() string {
	const unit = 1024
	if a.ContentLength < unit {
		return fmt.Sprintf("%d Б", a.ContentLength)
	}
	units := []string{"КБ", "МБ", "ГБ"}
	div, exp := int64(unit), 0
	for n := a.ContentLength / unit; n >= unit && exp < len(units)-1; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(a.ContentLength)/float64(div), units[exp])
}

// Event — встреча календаря (FindItem + CalendarView; повторяющиеся уже
// раскрыты сервером в экземпляры).
type Event struct {
	ItemID            string
	ChangeKey         string
	Subject           string
	Start, End        time.Time
	Location          string
	CalendarItemType  string // Single | Occurrence | Exception
	Organizer         Address
	MyResponseType    string // Accept | Tentative | Decline | NoResponseReceived | Organizer | Unknown
	RequiredAttendees []Address
	OptionalAttendees []Address
}

// Duration возвращает длительность встречи.
func (e Event) Duration() time.Duration { return e.End.Sub(e.Start) }

// ResponseTypeLabel переводит MyResponseType в русский ярлык для UI.
func ResponseTypeLabel(responseType string) string {
	switch strings.TrimSpace(responseType) {
	case "Accept":
		return "принято"
	case "Tentative":
		return "под вопросом"
	case "Decline":
		return "отклонено"
	case "NoResponseReceived":
		return "нет ответа"
	case "Organizer":
		return "организатор"
	default:
		return "неизвестно"
	}
}
