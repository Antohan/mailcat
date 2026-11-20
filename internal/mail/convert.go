package mail

// Конвертация структур internal/ews (SOAP-близкие) в доменные модели
// internal/mail. Единственное место, где пакеты соприкасаются: ews ничего не
// знает о mail (архитектура — docs/ARCHITECTURE.md §1, поток данных).

import (
	"mailcat/internal/ews"
)

// Folders конвертирует список папок из FindFolder.
func Folders(list []ews.Folder) []Folder {
	if len(list) == 0 {
		return nil
	}
	out := make([]Folder, 0, len(list))
	for _, f := range list {
		out = append(out, Folder{
			ID:               f.ID,
			ChangeKey:        f.ChangeKey,
			DisplayName:      f.DisplayName,
			TotalCount:       f.TotalCount,
			UnreadCount:      f.UnreadCount,
			ChildFolderCount: f.ChildFolderCount,
		})
	}
	return out
}

// MessageSummaries конвертирует страницу списка писем из FindItem.
func MessageSummaries(list []ews.MessageRef) []MessageSummary {
	if len(list) == 0 {
		return nil
	}
	out := make([]MessageSummary, 0, len(list))
	for _, m := range list {
		out = append(out, MessageSummary{
			ItemID:         m.ItemID,
			ChangeKey:      m.ChangeKey,
			Subject:        m.Subject,
			Received:       m.Received,
			IsRead:         m.IsRead,
			HasAttachments: m.HasAttachments,
			From:           Address{Name: m.FromName, Email: m.FromEmail},
		})
	}
	return out
}

// FullMessage конвертирует полное письмо из GetItem. Текст тела: приоритет у
// TextBody сервера; если сервер вернул только HTML — конвертация (оффлайн).
func FullMessage(d ews.MessageDetail) Message {
	bodyText := d.BodyText
	if bodyText == "" && d.BodyHTML != "" {
		bodyText = HTMLToText(d.BodyHTML)
	}
	return Message{
		MessageSummary: MessageSummary{
			ItemID:    d.ItemID,
			ChangeKey: d.ChangeKey,
			Subject:   d.Subject,
			Received:  d.Sent,
			From:      Address{},
		},
		Sent:              d.Sent,
		InternetMessageID: d.InternetMessageID,
		BodyHTML:          d.BodyHTML,
		BodyText:          bodyText,
		To:                mailboxesToAddresses(d.To),
		Cc:                mailboxesToAddresses(d.Cc),
		Attachments:       attachments(d.Attachments),
	}
}

// Events конвертирует встречи из CalendarView.
func Events(list []ews.CalendarEvent) []Event {
	if len(list) == 0 {
		return nil
	}
	out := make([]Event, 0, len(list))
	for _, e := range list {
		out = append(out, Event{
			ItemID:            e.ItemID,
			ChangeKey:         e.ChangeKey,
			Subject:           e.Subject,
			Start:             e.Start,
			End:               e.End,
			Location:          e.Location,
			CalendarItemType:  e.ItemType,
			MyResponseType:    e.MyResponseType,
			Organizer:         Address{Name: e.Organizer.Name, Email: e.Organizer.Email},
			RequiredAttendees: mailboxesToAddresses(e.Required),
			OptionalAttendees: mailboxesToAddresses(e.Optional),
		})
	}
	return out
}

func mailboxesToAddresses(list []ews.Mailbox) []Address {
	if len(list) == 0 {
		return nil
	}
	out := make([]Address, 0, len(list))
	for _, mb := range list {
		out = append(out, Address{Name: mb.Name, Email: mb.Email})
	}
	return out
}

func attachments(list []ews.AttachmentMeta) []Attachment {
	if len(list) == 0 {
		return nil
	}
	out := make([]Attachment, 0, len(list))
	for _, a := range list {
		out = append(out, Attachment{
			AttachmentID:  a.AttachmentID,
			Name:          a.Name,
			ContentType:   a.ContentType,
			ContentLength: a.ContentLength,
		})
	}
	return out
}
