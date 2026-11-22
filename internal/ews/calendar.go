package ews

// FindItem + CalendarView — встречи за диапазон дат из папки calendar
// (docs/EWS-REFERENCE.md §8). CalendarView сам раскрывает повторяющиеся
// встречи в экземпляры; пагинации нет (без IndexedPageItemView).
// Read-only (AGENTS.md §1).

import (
	"context"
	"encoding/xml"
	"errors"
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

// xmlCalendarItem — t:CalendarItem из ответа CalendarView (§8) или GetItem
// описания (§11): тело запрашивается только у GetItem.
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
	Body           xmlBody           `xml:"Body"`
	TextBody       string            `xml:"TextBody"`
}

// EventBody — описание встречи из GetItem (§11). ItemId и ChangeKey уже есть
// из CalendarView (§8), поэтому в ответе они не дублируются.
type EventBody struct {
	BodyType string // Html | Text — формат t:Body из ответа (fallback-путь)
	BodyHTML string // содержимое t:Body при BodyType="Html"
	BodyText string // t:TextBody; если сервер вернул Text — t:Body
}

// GetEventBody читает описание встречи по ItemId и ChangeKey (EWS-REFERENCE §11).
func (c *Client) GetEventBody(ctx context.Context, itemID, changeKey string) (EventBody, error) {
	var resp getEventBodyResponse
	if err := c.Do(ctx, newEventBodyRequest(itemID, changeKey), &resp); err != nil {
		return EventBody{}, fmt.Errorf("чтение встречи: %w", err)
	}
	body, err := resp.result()
	if err != nil {
		return EventBody{}, fmt.Errorf("чтение встречи: %w", err)
	}
	return body, nil
}

// newEventBodyRequest строит запрос по шаблону EWS-REFERENCE.md §11: тела нет
// в BaseShape, нужен item:TextBody; BodyType не запрашиваем (§6/§11).
func newEventBodyRequest(itemID, changeKey string) *getItemRequest {
	return &getItemRequest{
		ItemShape: itemShapeRequest{
			BaseShape: "Default",
			AdditionalProperties: &additionalPropertiesRequest{FieldURIs: []fieldURIRequest{
				{FieldURI: "item:TextBody"},
			}},
		},
		ItemIDs: itemIdsRequest{
			ItemID: xmlItemIDRequest{ID: itemID, ChangeKey: changeKey},
		},
	}
}

// getEventBodyResponse — m:GetItemResponse с t:CalendarItem (разбор по
// локальным именам, soap.go); отдельная структура от письма — другой тип
// элемента в t:Items.
type getEventBodyResponse struct {
	XMLName          xml.Name `xml:"GetItemResponse"`
	ResponseMessages struct {
		GetItemResponseMessage struct {
			responseHeader
			Items struct {
				CalendarItem []xmlCalendarItem `xml:"CalendarItem"`
			} `xml:"Items"`
		} `xml:"GetItemResponseMessage"`
	} `xml:"ResponseMessages"`
}

// result проверяет ResponseClass и конвертирует первое описание встречи.
func (r *getEventBodyResponse) result() (EventBody, error) {
	msg := &r.ResponseMessages.GetItemResponseMessage
	if err := msg.err(); err != nil {
		return EventBody{}, err
	}
	if len(msg.Items.CalendarItem) == 0 {
		return EventBody{}, errors.New("в ответе GetItem нет встречи (удалена или перемещена)")
	}
	return msg.Items.CalendarItem[0].eventBody(), nil
}

// eventBody конвертирует t:CalendarItem в EventBody (та же логика, что у
// письма: TextBody основной, t:Body — fallback).
func (it *xmlCalendarItem) eventBody() EventBody {
	b := EventBody{BodyType: it.Body.BodyType, BodyText: it.TextBody}
	if it.Body.BodyType == "Html" {
		b.BodyHTML = it.Body.Value
	} else if b.BodyText == "" {
		b.BodyText = it.Body.Value
	}
	return b
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
