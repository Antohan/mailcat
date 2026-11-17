package ews

// Тесты GetItem (docs/EWS-REFERENCE.md §6): golden-запрос, разбор письма
// с HTML-телом, TextBody, получателями и метаданными вложений.

import (
	"encoding/xml"
	"testing"
	"time"
)

// TestGetItemRequestGolden: запрос m:GetItem структурно совпадает с шаблоном
// §6 (Default + BodyType Best + пять AdditionalProperties + ItemId c ChangeKey).
func TestGetItemRequestGolden(t *testing.T) {
	req := newGetItemRequest("AAMkAGI2MDE=", "CQAAABYA0AA=")
	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса GetItem: %v", err)
	}
	want := []byte(`<m:GetItem
    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:BodyType BodyType="Best"/>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:TextBody"/>
      <t:FieldURI FieldURI="message:InternetMessageId"/>
      <t:FieldURI FieldURI="message:ToRecipients"/>
      <t:FieldURI FieldURI="message:CcRecipients"/>
      <t:FieldURI FieldURI="item:DateTimeSent"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:ItemIds>
    <t:ItemId Id="AAMkAGI2MDE=" ChangeKey="CQAAABYA0AA="/>
  </m:ItemIds>
</m:GetItem>`)
	assertXMLLike(t, "запрос GetItem", data, want)
}

// TestGetMessageParse: фикстура getitem_response.xml → MessageDetail с телом,
// получателями и двумя вложениями.
func TestGetMessageParse(t *testing.T) {
	var resp getItemResponse
	if err := parseResponse(fixture(t, "getitem_response.xml"), &resp); err != nil {
		t.Fatalf("разбор ответа GetItem: %v", err)
	}
	detail, err := resp.result()
	if err != nil {
		t.Fatalf("result(): %v", err)
	}
	want := MessageDetail{
		ItemID:            "AAMkAGI2MDE=",
		ChangeKey:         "CQAAABYA0AA=",
		Subject:           "Отчёт Q3",
		Sent:              time.Date(2026, 11, 17, 10, 12, 0, 0, time.UTC),
		BodyType:          "Html",
		BodyHTML:          "<html><body><p>Отчёт за Q3 готов. Детали во вложении.</p></body></html>",
		BodyText:          "Отчёт за Q3 готов. Детали во вложении.",
		InternetMessageID: "<CAFxKQ9Jx7@mail.company.example>",
		To: []Mailbox{
			{Name: "Петрова А.", Email: "apetrova@company.example"},
			{Name: "Сидоров В.", Email: "vsidorov@company.example"},
		},
		Cc: []Mailbox{
			{Name: "Кузнецов Д.", Email: "dkuznetsov@company.example"},
		},
		Attachments: []AttachmentMeta{
			{AttachmentID: "AAMkAGI2ATT1=", Name: "report.pdf", ContentType: "application/pdf", ContentLength: 2097152},
			{AttachmentID: "AAMkAGI2ATT2=", Name: "данные.csv", ContentType: "text/csv", ContentLength: 1024},
		},
	}
	// сравнение по полям: в detail есть срезы, == не подходит
	if detail.ItemID != want.ItemID || detail.ChangeKey != want.ChangeKey ||
		detail.Subject != want.Subject || !detail.Sent.Equal(want.Sent) ||
		detail.BodyType != want.BodyType || detail.BodyHTML != want.BodyHTML ||
		detail.BodyText != want.BodyText || detail.InternetMessageID != want.InternetMessageID {
		t.Errorf("detail = %+v, хочу %+v", detail, want)
	}
	if len(detail.To) != 2 || detail.To[0] != want.To[0] || detail.To[1] != want.To[1] {
		t.Errorf("To = %+v, хочу %+v", detail.To, want.To)
	}
	if len(detail.Cc) != 1 || detail.Cc[0] != want.Cc[0] {
		t.Errorf("Cc = %+v, хочу %+v", detail.Cc, want.Cc)
	}
	if len(detail.Attachments) != 2 {
		t.Fatalf("вложений %d, хочу 2", len(detail.Attachments))
	}
	for i, w := range want.Attachments {
		if detail.Attachments[i] != w {
			t.Errorf("вложение[%d] = %+v, хочу %+v", i, detail.Attachments[i], w)
		}
	}
}

// TestGetMessageParseTextBody: сервер вернул лучшим текстовый Body и без
// TextBody — BodyText берётся из t:Body, BodyHTML пуст.
func TestGetMessageParseTextBody(t *testing.T) {
	doc := []byte(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
               xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <m:GetItemResponse>
      <m:ResponseMessages>
        <m:GetItemResponseMessage ResponseClass="Success">
          <m:ResponseCode>NoError</m:ResponseCode>
          <m:Items>
            <t:Message>
              <t:ItemId Id="AAMkAGI2MDk=" ChangeKey="CQAAABYA0Ak="/>
              <t:Subject>Простое письмо</t:Subject>
              <t:Body BodyType="Text">Просто текст без HTML.</t:Body>
            </t:Message>
          </m:Items>
        </m:GetItemResponseMessage>
      </m:ResponseMessages>
    </m:GetItemResponse>
  </soap:Body>
</soap:Envelope>`)
	var resp getItemResponse
	if err := parseResponse(doc, &resp); err != nil {
		t.Fatalf("разбор ответа GetItem: %v", err)
	}
	detail, err := resp.result()
	if err != nil {
		t.Fatalf("result(): %v", err)
	}
	if detail.BodyType != "Text" || detail.BodyHTML != "" || detail.BodyText != "Просто текст без HTML." {
		t.Errorf("текстовое письмо разобрано неверно: %+v", detail)
	}
}
