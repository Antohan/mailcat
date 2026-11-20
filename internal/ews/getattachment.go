package ews

// GetAttachment — скачивание одного вложения по AttachmentId
// (docs/EWS-REFERENCE.md §7). Лимит размера проверяется ДО отправки запроса
// (docs/SECURITY.md §5). Read-only (AGENTS.md §1).

import (
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
)

// GetAttachment возвращает байты вложения. Если att.ContentLength известен
// (> 0) и превышает maxBytes, скачивание отклоняется до запроса. На саму
// операцию отводится увеличенный таймаут 5 минут (SECURITY.md §7).
func (c *Client) GetAttachment(ctx context.Context, att AttachmentMeta, maxBytes int64) ([]byte, error) {
	if att.ContentLength > 0 && att.ContentLength > maxBytes {
		return nil, fmt.Errorf("вложение %s: размер %d байт превышает лимит %d — скачивание отклонено",
			att.Name, att.ContentLength, maxBytes)
	}
	ctx, cancel := context.WithTimeout(ctx, attachmentTimeout)
	defer cancel()

	var resp getAttachmentResponse
	if err := c.Do(ctx, newGetAttachmentRequest(att.AttachmentID), &resp); err != nil {
		return nil, fmt.Errorf("скачивание вложения %s: %w", att.Name, err)
	}
	data, err := resp.result()
	if err != nil {
		return nil, fmt.Errorf("скачивание вложения %s: %w", att.Name, err)
	}
	// Размер был неизвестен (ContentLength=0) — проверяем по факту, до выдачи
	// данных наружу (security-ревью этапа 1, §5).
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("вложение %s: скачанный размер %d байт превышает лимит %d — отклонено",
			att.Name, len(data), maxBytes)
	}
	return data, nil
}

// newGetAttachmentRequest строит запрос по шаблону EWS-REFERENCE.md §7.
func newGetAttachmentRequest(attachmentID string) *getAttachmentRequest {
	return &getAttachmentRequest{
		AttachmentShape: attachmentShapeRequest{BaseShape: "IdOnly"},
		AttachmentIDs: attachmentIdsRequest{
			AttachmentID: attachmentIDRequest{ID: attachmentID},
		},
	}
}

type getAttachmentRequest struct {
	XMLName         xml.Name               `xml:"http://schemas.microsoft.com/exchange/services/2006/messages GetAttachment"`
	AttachmentShape attachmentShapeRequest `xml:"AttachmentShape"`
	AttachmentIDs   attachmentIdsRequest   `xml:"AttachmentIds"`
}

// attachmentShapeRequest — m:AttachmentShape с t:BaseShape (namespace BaseShape
// задаётся в теге поля — сам элемент наследует messages от родителя).
type attachmentShapeRequest struct {
	BaseShape string `xml:"http://schemas.microsoft.com/exchange/services/2006/types BaseShape"`
}

type attachmentIdsRequest struct {
	XMLName      xml.Name            `xml:"http://schemas.microsoft.com/exchange/services/2006/messages AttachmentIds"`
	AttachmentID attachmentIDRequest `xml:"http://schemas.microsoft.com/exchange/services/2006/types AttachmentId"`
}

// attachmentIDRequest — t:AttachmentId@Id в запросе.
type attachmentIDRequest struct {
	ID string `xml:"Id,attr"`
}

// getAttachmentResponse — m:GetAttachmentResponse (разбор по локальным именам).
type getAttachmentResponse struct {
	XMLName          xml.Name `xml:"GetAttachmentResponse"`
	ResponseMessages struct {
		GetAttachmentResponseMessage struct {
			responseHeader
			Attachments struct {
				FileAttachment []xmlAttachmentContent `xml:"FileAttachment"`
			} `xml:"Attachments"`
		} `xml:"GetAttachmentResponseMessage"`
	} `xml:"ResponseMessages"`
}

// xmlAttachmentContent — t:FileAttachment с base64-контентом (§7).
type xmlAttachmentContent struct {
	AttachmentID xmlAttachmentIDRef `xml:"AttachmentId"`
	Content      string             `xml:"Content"`
}

// result проверяет ResponseClass и декодирует содержимое первого вложения.
func (r *getAttachmentResponse) result() ([]byte, error) {
	msg := &r.ResponseMessages.GetAttachmentResponseMessage
	if err := msg.err(); err != nil {
		return nil, err
	}
	if len(msg.Attachments.FileAttachment) == 0 {
		return nil, fmt.Errorf("в ответе GetAttachment нет вложения")
	}
	data, err := base64.StdEncoding.DecodeString(msg.Attachments.FileAttachment[0].Content)
	if err != nil {
		return nil, fmt.Errorf("декодирование base64 содержимого вложения: %w", err)
	}
	return data, nil
}
