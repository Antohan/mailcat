package ews

// GetItem — чтение одного письма (docs/EWS-REFERENCE.md §6). Содержимое
// вложений НЕ приходит — только метаданные (блоб качает GetAttachment, §7).
// Read-only (AGENTS.md §1).

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"
)

// Mailbox — корреспондент/участник: отображаемое имя + e-mail.
type Mailbox struct {
	Name  string
	Email string
}

// AttachmentMeta — метаданные вложения из GetItem.
type AttachmentMeta struct {
	AttachmentID  string
	Name          string
	ContentType   string
	ContentLength int64 // байт; проверяется против лимита в GetAttachment
}

// MessageDetail — полное письмо (GetItem).
type MessageDetail struct {
	ItemID            string
	ChangeKey         string
	Subject           string
	Sent              time.Time
	BodyType          string // Html | Text — формат t:Body из ответа (fallback-путь)
	BodyHTML          string // содержимое t:Body при BodyType="Html"
	BodyText          string // t:TextBody; если сервер вернул Text — t:Body
	InternetMessageID string
	To                []Mailbox
	Cc                []Mailbox
	Attachments       []AttachmentMeta
}

// GetMessage возвращает полное письмо по ItemId и ChangeKey (ChangeKey обязателен:
// без него сервер может отдать другую версию письма).
func (c *Client) GetMessage(ctx context.Context, itemID, changeKey string) (MessageDetail, error) {
	var resp getItemResponse
	if err := c.Do(ctx, newGetItemRequest(itemID, changeKey), &resp); err != nil {
		return MessageDetail{}, fmt.Errorf("чтение письма: %w", err)
	}
	detail, err := resp.result()
	if err != nil {
		return MessageDetail{}, fmt.Errorf("чтение письма: %w", err)
	}
	return detail, nil
}

// newGetItemRequest строит запрос по шаблону EWS-REFERENCE.md §6.
func newGetItemRequest(itemID, changeKey string) *getItemRequest {
	return &getItemRequest{
		ItemShape: itemShapeRequest{
			// WHY: без t:BodyType@BodyType="Best".
			// Why: живой Exchange 2019 (15.2.2562) на BodyType в ItemShape
			//  отвечает 500 ErrorInvalidRequest (проверено изолированно);
			//  BodyType управляет только форматом t:Body, который мы не
			//  запрашиваем — TextBody от него не зависит (вердикт ews-спекта).
			// Not to do: вернуть BodyType — живой сервер его отвергает.
			BaseShape: "Default",
			AdditionalProperties: &additionalPropertiesRequest{FieldURIs: []fieldURIRequest{
				{FieldURI: "item:TextBody"},
				{FieldURI: "message:InternetMessageId"},
				{FieldURI: "message:ToRecipients"},
				{FieldURI: "message:CcRecipients"},
				{FieldURI: "item:DateTimeSent"},
			}},
		},
		ItemIDs: itemIdsRequest{
			ItemID: xmlItemIDRequest{ID: itemID, ChangeKey: changeKey},
		},
	}
}

type getItemRequest struct {
	XMLName   xml.Name         `xml:"http://schemas.microsoft.com/exchange/services/2006/messages GetItem"`
	ItemShape itemShapeRequest `xml:"ItemShape"`
	ItemIDs   itemIdsRequest   `xml:"ItemIds"`
}

type itemIdsRequest struct {
	XMLName xml.Name `xml:"http://schemas.microsoft.com/exchange/services/2006/messages ItemIds"`
	// t:ItemId — namespace types, не родительский messages (EWS-REFERENCE.md §6).
	ItemID xmlItemIDRequest `xml:"http://schemas.microsoft.com/exchange/services/2006/types ItemId"`
}

// xmlItemIDRequest — t:ItemId@Id,@ChangeKey в запросах.
type xmlItemIDRequest struct {
	ID        string `xml:"Id,attr"`
	ChangeKey string `xml:"ChangeKey,attr,omitempty"`
}

// getItemResponse — m:GetItemResponse (разбор по локальным именам, soap.go).
type getItemResponse struct {
	XMLName          xml.Name `xml:"GetItemResponse"`
	ResponseMessages struct {
		GetItemResponseMessage struct {
			responseHeader
			Items struct {
				Message []xmlMessage `xml:"Message"`
			} `xml:"Items"`
		} `xml:"GetItemResponseMessage"`
	} `xml:"ResponseMessages"`
}

// result проверяет ResponseClass и возвращает первое письмо ответа.
func (r *getItemResponse) result() (MessageDetail, error) {
	msg := &r.ResponseMessages.GetItemResponseMessage
	if err := msg.err(); err != nil {
		return MessageDetail{}, err
	}
	if len(msg.Items.Message) == 0 {
		return MessageDetail{}, errors.New("в ответе GetItem нет письма (письмо удалено или перемещено)")
	}
	return msg.Items.Message[0].detail(), nil
}

// detail конвертирует t:Message в MessageDetail.
func (m *xmlMessage) detail() MessageDetail {
	d := MessageDetail{
		ItemID:            m.ItemID.ID,
		ChangeKey:         m.ItemID.ChangeKey,
		Subject:           m.Subject,
		Sent:              m.DateTimeSent,
		BodyType:          m.Body.BodyType,
		BodyText:          m.TextBody,
		InternetMessageID: m.InternetMessageID,
	}
	switch m.Body.BodyType {
	case "Html":
		d.BodyHTML = m.Body.Value
	default:
		// Text (или сервер не вернул Body): если готового TextBody нет,
		// текстовым телом считается содержимое t:Body
		if d.BodyText == "" {
			d.BodyText = m.Body.Value
		}
	}
	d.To = mailboxes(m.ToRecipients)
	d.Cc = mailboxes(m.CcRecipients)
	if m.Attachments != nil {
		d.Attachments = make([]AttachmentMeta, 0, len(m.Attachments.FileAttachment))
		for _, a := range m.Attachments.FileAttachment {
			d.Attachments = append(d.Attachments, AttachmentMeta{
				AttachmentID:  a.AttachmentID.ID,
				Name:          a.Name,
				ContentType:   a.ContentType,
				ContentLength: a.ContentLength,
			})
		}
	}
	return d
}

// mailboxes конвертирует список t:Mailbox в []Mailbox.
func mailboxes(list *xmlMailboxList) []Mailbox {
	if list == nil || len(list.Mailbox) == 0 {
		return nil
	}
	out := make([]Mailbox, 0, len(list.Mailbox))
	for _, mb := range list.Mailbox {
		out = append(out, Mailbox{Name: mb.Name, Email: mb.EmailAddress})
	}
	return out
}
