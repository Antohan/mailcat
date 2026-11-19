package ews

// FindItem — список писем с постраничной пагинацией (docs/EWS-REFERENCE.md §5).
// Общие примитивы запросов (ItemShape, SortOrder, views) живут здесь и
// переиспользуются GetItem и CalendarView. Read-only (AGENTS.md §1).

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"
)

// MessageRef — письмо в списке (FindItem): ровно то, что нужно строке списка.
type MessageRef struct {
	ItemID         string
	ChangeKey      string
	Subject        string
	Received       time.Time
	IsRead         bool
	HasAttachments bool
	FromName       string
	FromEmail      string
}

// Page — одна страница FindItem с курсором пагинации сервера.
type Page struct {
	Items        []MessageRef
	NextOffset   int  // IndexedPagingOffset — Offset следующего запроса
	IncludesLast bool // IncludesLastItemInRange — дальше страниц нет
	TotalInView  int  // TotalItemsInView
}

// NextOffset возвращает Offset следующей страницы по правилам
// EWS-REFERENCE.md §5: Offset = IndexedPagingOffset прошлого ответа, стоп
// при IncludesLastItemInRange=true (второе значение false — листать больше
// не нужно).
func NextOffset(p Page) (int, bool) {
	if p.IncludesLast {
		return 0, false
	}
	return p.NextOffset, true
}

// FindMessages возвращает страницу писем обычной папки (FolderId из
// FindFolder), новые сверху (SortOrder Descending по DateTimeReceived).
func (c *Client) FindMessages(ctx context.Context, folder FolderID, pageSize, offset int) (Page, error) {
	req, err := newFindMessagesRequest(parentFolderIdsRequest{
		FolderID: &folderIDRequest{ID: folder.ID, ChangeKey: folder.ChangeKey},
	}, pageSize, offset)
	if err != nil {
		return Page{}, fmt.Errorf("список писем: %w", err)
	}
	return c.doFindMessages(ctx, req)
}

// FindDistinguishedMessages — то же для distinguished-папки (inbox, sentitems,
// ... — EWS-REFERENCE.md §9).
func (c *Client) FindDistinguishedMessages(ctx context.Context, folderID string, pageSize, offset int) (Page, error) {
	req, err := newFindMessagesRequest(parentFolderIdsRequest{
		DistinguishedFolderID: &distinguishedFolderIDRequest{ID: folderID},
	}, pageSize, offset)
	if err != nil {
		return Page{}, fmt.Errorf("список писем: %w", err)
	}
	return c.doFindMessages(ctx, req)
}

// doFindMessages отправляет запрос и превращает ответ в Page.
func (c *Client) doFindMessages(ctx context.Context, req *findItemRequest) (Page, error) {
	var resp findItemResponse
	if err := c.Do(ctx, req, &resp); err != nil {
		return Page{}, fmt.Errorf("список писем: %w", err)
	}
	page, err := resp.result()
	if err != nil {
		return Page{}, fmt.Errorf("список писем: %w", err)
	}
	return page, nil
}

// newFindMessagesRequest строит запрос по шаблону EWS-REFERENCE.md §5.
func newFindMessagesRequest(parent parentFolderIdsRequest, pageSize, offset int) (*findItemRequest, error) {
	if pageSize < 1 {
		return nil, fmt.Errorf("некорректный размер страницы: %d (нужно ≥ 1)", pageSize)
	}
	if offset < 0 {
		return nil, fmt.Errorf("некорректный отступ страницы: %d (нужно ≥ 0)", offset)
	}
	return &findItemRequest{
		Traversal: "Shallow",
		ItemShape: itemShapeRequest{
			// WHY: BaseShape Default, а не IdOnly со списком FieldURI.
			// Why: живой Exchange 2019 (15.2.2562) на item:IsRead в
			//  AdditionalProperties отвечает 500 ErrorInvalidRequest; Default
			//  возвращает Subject/IsRead/From/HasAttachments сам, но без
			//  DateTimeReceived — его добираем единственным FieldURI (он прошёл).
			// Not to do: вернуть item:IsRead в AdditionalProperties — проверено, 500.
			BaseShape: "Default",
			AdditionalProperties: &additionalPropertiesRequest{FieldURIs: []fieldURIRequest{
				{FieldURI: "item:DateTimeReceived"},
			}},
		},
		IndexedPageItemView: &indexedPageItemViewRequest{
			MaxEntriesReturned: pageSize,
			Offset:             offset,
			BasePoint:          "Beginning",
		},
		SortOrder: sortOrderRequest{FieldOrders: []fieldOrderRequest{{
			Order:    "Descending",
			FieldURI: fieldURIRequest{FieldURI: "item:DateTimeReceived"},
		}}},
		ParentFolderIDs: parent,
	}, nil
}

// findItemRequest — m:FindItem: постраничный список писем (§5) или CalendarView
// (§8); в одном запросе живёт ровно одна вьюха (IndexedPageItemView XOR
// CalendarView) — лишняя опускается (nil + omitempty).
type findItemRequest struct {
	XMLName             xml.Name                    `xml:"http://schemas.microsoft.com/exchange/services/2006/messages FindItem"`
	Traversal           string                      `xml:"Traversal,attr"`
	ItemShape           itemShapeRequest            `xml:"ItemShape"`
	IndexedPageItemView *indexedPageItemViewRequest `xml:",omitempty"`
	CalendarView        *calendarViewRequest        `xml:",omitempty"`
	SortOrder           sortOrderRequest            `xml:"SortOrder"`
	ParentFolderIDs     parentFolderIdsRequest      `xml:"ParentFolderIds"`
}

// itemShapeRequest — m:ItemShape: BaseShape + опциональные BodyType и
// AdditionalProperties (используется FindItem, GetItem, CalendarView).
type itemShapeRequest struct {
	BaseShape            string                       `xml:"http://schemas.microsoft.com/exchange/services/2006/types BaseShape"`
	BodyType             *bodyTypeRequest             `xml:",omitempty"`
	AdditionalProperties *additionalPropertiesRequest `xml:",omitempty"`
}

// bodyTypeRequest — t:BodyType@BodyType="Best" (GetItem, §6).
type bodyTypeRequest struct {
	XMLName  xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/types BodyType"`
	BodyType string   `xml:"BodyType,attr"`
}

// additionalPropertiesRequest — t:AdditionalProperties со списком FieldURI
// (только URI из EWS-REFERENCE.md §9).
type additionalPropertiesRequest struct {
	XMLName   xml.Name          `xml:"http://schemas.microsoft.com/exchange/services/2006/types AdditionalProperties"`
	FieldURIs []fieldURIRequest `xml:"FieldURI"`
}

type fieldURIRequest struct {
	FieldURI string `xml:"FieldURI,attr"`
}

// indexedPageItemViewRequest — m:IndexedPageItemView (только без CalendarView).
type indexedPageItemViewRequest struct {
	XMLName            xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/messages IndexedPageItemView"`
	MaxEntriesReturned int      `xml:"MaxEntriesReturned,attr"`
	Offset             int      `xml:"Offset,attr"`
	BasePoint          string   `xml:"BasePoint,attr"`
}

// calendarViewRequest — m:CalendarView@StartDate,@EndDate (RFC3339, UTC)
// @MaxEntriesReturned (§8); пагинации нет.
type calendarViewRequest struct {
	XMLName            xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/messages CalendarView"`
	StartDate          string   `xml:"StartDate,attr"`
	EndDate            string   `xml:"EndDate,attr"`
	MaxEntriesReturned int      `xml:"MaxEntriesReturned,attr"`
}

// sortOrderRequest — m:SortOrder со списком t:FieldOrder.
type sortOrderRequest struct {
	XMLName     xml.Name            `xml:"http://schemas.microsoft.com/exchange/services/2006/messages SortOrder"`
	FieldOrders []fieldOrderRequest `xml:"FieldOrder"`
}

type fieldOrderRequest struct {
	XMLName  xml.Name        `xml:"http://schemas.microsoft.com/exchange/services/2006/types FieldOrder"`
	Order    string          `xml:"Order,attr"`
	FieldURI fieldURIRequest `xml:"FieldURI"`
}

// findItemResponse — m:FindItemResponse (разбор по локальным именам, soap.go).
type findItemResponse struct {
	XMLName          xml.Name `xml:"FindItemResponse"`
	ResponseMessages struct {
		FindItemResponseMessage findItemResponseMessage `xml:"FindItemResponseMessage"`
	} `xml:"ResponseMessages"`
}

type findItemResponseMessage struct {
	responseHeader
	RootFolder findItemRootFolder `xml:"RootFolder"`
}

type findItemRootFolder struct {
	IndexedPagingOffset     int       `xml:"IndexedPagingOffset,attr"`
	TotalItemsInView        int       `xml:"TotalItemsInView,attr"`
	IncludesLastItemInRange bool      `xml:"IncludesLastItemInRange,attr"`
	Items                   findItems `xml:"Items"`
}

type findItems struct {
	Message []xmlMessage `xml:"Message"`
}

// xmlMessage — t:Message: полный набор полей FindItem (§5) и GetItem (§6);
// отсутствующие в конкретном ответе элементы остаются нулевыми.
type xmlMessage struct {
	ItemID            xmlItemIDRef       `xml:"ItemId"`
	Subject           string             `xml:"Subject"`
	DateTimeReceived  time.Time          `xml:"DateTimeReceived"`
	DateTimeSent      time.Time          `xml:"DateTimeSent"`
	IsRead            bool               `xml:"IsRead"`
	HasAttachments    bool               `xml:"HasAttachments"`
	From              *xmlSingleMailbox  `xml:"From"`
	Body              xmlBody            `xml:"Body"`
	TextBody          string             `xml:"TextBody"`
	InternetMessageID string             `xml:"InternetMessageId"`
	ToRecipients      *xmlMailboxList    `xml:"ToRecipients"`
	CcRecipients      *xmlMailboxList    `xml:"CcRecipients"`
	Attachments       *xmlAttachmentList `xml:"Attachments"`
}

// xmlItemIDRef — t:ItemId@Id,@ChangeKey в ответах.
type xmlItemIDRef struct {
	ID        string `xml:"Id,attr"`
	ChangeKey string `xml:"ChangeKey,attr"`
}

// xmlSingleMailbox — обёртка t:From/t:Organizer с одной t:Mailbox.
type xmlSingleMailbox struct {
	Mailbox *xmlMailbox `xml:"Mailbox"`
}

// xmlMailbox — t:Mailbox: имя + адрес.
type xmlMailbox struct {
	Name         string `xml:"Name"`
	EmailAddress string `xml:"EmailAddress"`
}

// xmlMailboxList — t:ToRecipients/t:CcRecipients/t:RequiredAttendees/
// t:OptionalAttendees — список t:Mailbox.
type xmlMailboxList struct {
	Mailbox []xmlMailbox `xml:"Mailbox"`
}

// xmlBody — t:Body@BodyType (Html|Text) с содержимым.
type xmlBody struct {
	BodyType string `xml:"BodyType,attr"`
	Value    string `xml:",chardata"`
}

// xmlAttachmentList — t:Attachments из GetItem: метаданные без содержимого.
type xmlAttachmentList struct {
	FileAttachment []xmlFileAttachment `xml:"FileAttachment"`
}

// xmlFileAttachment — t:FileAttachment (§6): AttachmentId, Name, ContentType,
// ContentLength; сам блоб приходит отдельной операцией GetAttachment (§7).
type xmlFileAttachment struct {
	AttachmentID  xmlAttachmentIDRef `xml:"AttachmentId"`
	Name          string             `xml:"Name"`
	ContentType   string             `xml:"ContentType"`
	ContentLength int64              `xml:"ContentLength"`
}

// xmlAttachmentIDRef — t:AttachmentId@Id в ответах.
type xmlAttachmentIDRef struct {
	ID string `xml:"Id,attr"`
}

// result проверяет ResponseClass и конвертирует ответ в Page.
func (r *findItemResponse) result() (Page, error) {
	msg := &r.ResponseMessages.FindItemResponseMessage
	if err := msg.err(); err != nil {
		return Page{}, err
	}
	root := &msg.RootFolder
	items := make([]MessageRef, 0, len(root.Items.Message))
	for _, m := range root.Items.Message {
		ref := MessageRef{
			ItemID:         m.ItemID.ID,
			ChangeKey:      m.ItemID.ChangeKey,
			Subject:        m.Subject,
			Received:       m.DateTimeReceived,
			IsRead:         m.IsRead,
			HasAttachments: m.HasAttachments,
		}
		if m.From != nil && m.From.Mailbox != nil {
			ref.FromName = m.From.Mailbox.Name
			ref.FromEmail = m.From.Mailbox.EmailAddress
		}
		items = append(items, ref)
	}
	return Page{
		Items:        items,
		NextOffset:   root.IndexedPagingOffset,
		IncludesLast: root.IncludesLastItemInRange,
		TotalInView:  root.TotalItemsInView,
	}, nil
}
