package ews

// FindItem + CalendarView — встречи за диапазон дат из папки calendar
// (docs/EWS-REFERENCE.md §8). CalendarView сам раскрывает повторяющиеся
// встречи в экземпляры; пагинации нет (без IndexedPageItemView).
// Read-only (AGENTS.md §1).

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

// CalendarEvent — встреча календаря.
type CalendarEvent struct {
	ItemID         string
	ChangeKey      string
	Subject        string
	Start          time.Time
	End            time.Time
	Location       string
	ItemType       string // Single | Occurrence | Exception
	MyResponseType string // Accept | Tentative | Decline | NoResponseReceived | Organizer | Unknown
	Organizer      Mailbox
	Required       []Mailbox
	Optional       []Mailbox
}

// FindEvents возвращает встречи за период [start, end] (обычно месяц),
// отсортированные сервером по calendar:Start по возрастанию.
func (c *Client) FindEvents(ctx context.Context, start, end time.Time) ([]CalendarEvent, error) {
	if !end.After(start) {
		return nil, fmt.Errorf("некорректный диапазон календаря: конец %v не позже начала %v", end, start)
	}
	var resp findEventsResponse
	if err := c.Do(ctx, newFindEventsRequest(start, end), &resp); err != nil {
		return nil, fmt.Errorf("список встреч: %w", err)
	}
	events, err := resp.result()
	if err != nil {
		return nil, fmt.Errorf("список встреч: %w", err)
	}
	return events, nil
}

// newFindEventsRequest строит запрос по шаблону EWS-REFERENCE.md §8.
// Даты — xs:dateTime в UTC (справочник §8: «в запросе используем UTC»).
func newFindEventsRequest(start, end time.Time) *findItemRequest {
	return &findItemRequest{
		Traversal: "Shallow",
		ItemShape: itemShapeRequest{
			BaseShape: "Default",
			AdditionalProperties: &additionalPropertiesRequest{FieldURIs: []fieldURIRequest{
				{FieldURI: "calendar:Location"},
				{FieldURI: "calendar:Organizer"},
				{FieldURI: "calendar:MyResponseType"},
				{FieldURI: "calendar:RequiredAttendees"},
				{FieldURI: "calendar:OptionalAttendees"},
			}},
		},
		CalendarView: &calendarViewRequest{
			StartDate:          start.UTC().Format(time.RFC3339),
			EndDate:            end.UTC().Format(time.RFC3339),
			MaxEntriesReturned: 500,
		},
		// WHY: без SortOrder (и Restriction) — с CalendarView EWS их запрещает
		// (ErrorInvalidOperation, живой сервер 15.2.2562); ответ всегда идёт
		// по calendar:Start по возрастанию, порядок задаёт сам сервер.
		ParentFolderIDs: parentFolderIdsRequest{
			DistinguishedFolderID: &distinguishedFolderIDRequest{ID: "calendar"},
		},
	}
}

// findEventsResponse — m:FindItemResponse с t:CalendarItem (разбор по
// локальным именам; отдельная структура от findItemResponse — другой тип
// элементов в t:Items).
type findEventsResponse struct {
	XMLName          xml.Name `xml:"FindItemResponse"`
	ResponseMessages struct {
		FindItemResponseMessage struct {
			responseHeader
			RootFolder struct {
				Items struct {
					CalendarItem []xmlCalendarItem `xml:"CalendarItem"`
				} `xml:"Items"`
			} `xml:"RootFolder"`
		} `xml:"FindItemResponseMessage"`
	} `xml:"ResponseMessages"`
}

// xmlCalendarItem — t:CalendarItem из ответа CalendarView (§8).
type xmlCalendarItem struct {
	ItemID         xmlItemIDRef      `xml:"ItemId"`
	Subject        string            `xml:"Subject"`
	Start          time.Time         `xml:"Start"`
	End            time.Time         `xml:"End"`
	Location       string            `xml:"Location"`
	ItemType       string            `xml:"CalendarItemType"`
	Organizer      *xmlSingleMailbox `xml:"Organizer"`
	MyResponseType string            `xml:"MyResponseType"`
	Required       *xmlMailboxList   `xml:"RequiredAttendees"`
	Optional       *xmlMailboxList   `xml:"OptionalAttendees"`
}

// result проверяет ResponseClass и конвертирует ответ в []CalendarEvent.
func (r *findEventsResponse) result() ([]CalendarEvent, error) {
	msg := &r.ResponseMessages.FindItemResponseMessage
	if err := msg.err(); err != nil {
		return nil, err
	}
	items := msg.RootFolder.Items.CalendarItem
	events := make([]CalendarEvent, 0, len(items))
	for _, it := range items {
		ev := CalendarEvent{
			ItemID:         it.ItemID.ID,
			ChangeKey:      it.ItemID.ChangeKey,
			Subject:        it.Subject,
			Start:          it.Start,
			End:            it.End,
			Location:       it.Location,
			ItemType:       it.ItemType,
			MyResponseType: it.MyResponseType,
		}
		if it.Organizer != nil && it.Organizer.Mailbox != nil {
			ev.Organizer = Mailbox{Name: it.Organizer.Mailbox.Name, Email: it.Organizer.Mailbox.EmailAddress}
		}
		ev.Required = mailboxes(it.Required)
		ev.Optional = mailboxes(it.Optional)
		events = append(events, ev)
	}
	return events, nil
}
