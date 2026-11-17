package mail

// Тесты конвертации структур internal/ews в доменные модели internal/mail.
// Конвертеры живут здесь (mail импортирует ews; обратной зависимости нет) —
// ruling этапа 2: первый потребитель — UI.

import (
	"testing"
	"time"

	"mailcat/internal/ews"
)

func ts(t *testing.T, s string) time.Time {
	t.Helper()
	tv, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("парсинг времени %q: %v", s, err)
	}
	return tv
}

// TestConvertFolders: счётчики и идентификаторы переносятся без потерь.
func TestConvertFolders(t *testing.T) {
	got := Folders([]ews.Folder{
		{ID: "AQApA1", ChangeKey: "CK1", DisplayName: "Входящие", TotalCount: 1234, UnreadCount: 23, ChildFolderCount: 2},
		{ID: "AQApA2", ChangeKey: "CK2", DisplayName: "Отправленные", TotalCount: 890},
	})
	if len(got) != 2 {
		t.Fatalf("папок %d, хочу 2", len(got))
	}
	f := got[0]
	if f.ID != "AQApA1" || f.ChangeKey != "CK1" || f.DisplayName != "Входящие" ||
		f.TotalCount != 1234 || f.UnreadCount != 23 || f.ChildFolderCount != 2 {
		t.Errorf("папка сконвертирована неверно: %+v", f)
	}
	if got[1].UnreadCount != 0 {
		t.Errorf("UnreadCount по умолчанию: %+v", got[1])
	}
}

// TestConvertMessageRefs: From-строка становится Address; флаги и даты на месте.
func TestConvertMessageRefs(t *testing.T) {
	recv := ts(t, "2026-11-17T10:12:44Z")
	got := MessageSummaries([]ews.MessageRef{
		{ItemID: "I1", ChangeKey: "C1", Subject: "Отчёт Q3", Received: recv,
			IsRead: false, HasAttachments: true, FromName: "Иванов И.", FromEmail: "ivanov@company.example"},
	})
	if len(got) != 1 {
		t.Fatalf("писем %d, хочу 1", len(got))
	}
	m := got[0]
	if m.ItemID != "I1" || m.Subject != "Отчёт Q3" || !m.Received.Equal(recv) {
		t.Errorf("строка списка: %+v", m)
	}
	if m.IsRead || !m.HasAttachments {
		t.Errorf("флаги: %+v", m)
	}
	if m.From.Name != "Иванов И." || m.From.Email != "ivanov@company.example" {
		t.Errorf("From: %+v", m.From)
	}
}

// TestConvertMessageDetail: полный письмо: получатели, вложения, тело.
// BodyType=Html при пустом TextBody → HTML конвертируется в текст.
func TestConvertMessageDetail(t *testing.T) {
	sent := ts(t, "2026-11-17T10:12:00Z")
	detail := ews.MessageDetail{
		ItemID: "I1", ChangeKey: "C1", Subject: "Отчёт Q3", Sent: sent,
		BodyType: "Html",
		BodyHTML: "<p>Отчёт за Q3 готов.</p>",
		To:       []ews.Mailbox{{Name: "Петрова А.", Email: "apetrova@company.example"}},
		Cc:       []ews.Mailbox{{Name: "Кузнецов Д.", Email: "dkuznetsov@company.example"}},
		Attachments: []ews.AttachmentMeta{
			{AttachmentID: "A1", Name: "report.pdf", ContentType: "application/pdf", ContentLength: 2097152},
		},
	}
	got := FullMessage(detail)
	if got.Subject != "Отчёт Q3" || !got.Sent.Equal(sent) {
		t.Errorf("письмо: %+v", got)
	}
	if len(got.To) != 1 || got.To[0].Email != "apetrova@company.example" {
		t.Errorf("To: %+v", got.To)
	}
	if len(got.Cc) != 1 {
		t.Errorf("Cc: %+v", got.Cc)
	}
	if len(got.Attachments) != 1 || got.Attachments[0].Name != "report.pdf" || got.Attachments[0].ContentLength != 2097152 {
		t.Errorf("вложения: %+v", got.Attachments)
	}
	if got.BodyHTML == "" {
		t.Error("BodyHTML потерян")
	}
	if want := "Отчёт за Q3 готов."; got.BodyText != want {
		t.Errorf("BodyText = %q, хочу %q (конвертация HTML)", got.BodyText, want)
	}
}

// TestConvertMessageDetailTextBodyPreferred: сервер дал TextBody — конвертация
// HTML не запускается, TextBody используется как есть.
func TestConvertMessageDetailTextBodyPreferred(t *testing.T) {
	detail := ews.MessageDetail{
		BodyType: "Html",
		BodyHTML: "<p>HTML-версия</p>",
		BodyText: "Текстовая версия",
	}
	if got := FullMessage(detail); got.BodyText != "Текстовая версия" {
		t.Errorf("BodyText = %q, хочу текстовую версию сервера", got.BodyText)
	}
}

// TestConvertEvents: участники, организатор, статусы.
func TestConvertEvents(t *testing.T) {
	start := ts(t, "2026-11-17T10:00:00Z")
	end := ts(t, "2026-11-17T10:30:00Z")
	got := Events([]ews.CalendarEvent{
		{ItemID: "E1", ChangeKey: "K1", Subject: "Синк команды", Start: start, End: end,
			Location: "Zoom", ItemType: "Occurrence", MyResponseType: "Accept",
			Organizer: ews.Mailbox{Name: "Иванов И.", Email: "ivanov@company.example"},
			Required:  []ews.Mailbox{{Email: "apetrova@company.example"}},
			Optional:  []ews.Mailbox{{Email: "dkuznetsov@company.example"}},
		},
	})
	if len(got) != 1 {
		t.Fatalf("событий %d, хочу 1", len(got))
	}
	e := got[0]
	if e.Subject != "Синк команды" || !e.Start.Equal(start) || !e.End.Equal(end) {
		t.Errorf("событие: %+v", e)
	}
	if e.CalendarItemType != "Occurrence" || e.MyResponseType != "Accept" {
		t.Errorf("тип/статус: %+v", e)
	}
	if e.Organizer.Email != "ivanov@company.example" {
		t.Errorf("организатор: %+v", e.Organizer)
	}
	if len(e.RequiredAttendees) != 1 || len(e.OptionalAttendees) != 1 {
		t.Errorf("участники: %+v %+v", e.RequiredAttendees, e.OptionalAttendees)
	}
	if ResponseTypeLabel(e.MyResponseType) != "принято" {
		t.Errorf("ярлык статуса не работает в связке")
	}
}
