---
name: EWS Reference
description: Точные SOAP-шаблоны Exchange Web Services для mailcat — использовать при любой работе с EWS/XML, чтобы не выдумывать схему
---

# EWS Reference (компакт)

ПОЛНЫЙ справочник: `docs/EWS-REFERENCE.md` (repo root). Ниже — минимальные
скелеты для быстрой ориентировки. Правило: элемент/атрибут/FieldURI, которых
нет ни здесь, ни там — НЕ существует; уточняй у пользователя.

## Транспорт

`POST https://<server>/EWS/Exchange.asmx`, `Content-Type: text/xml; charset=utf-8`,
Basic auth; при 401 + `WWW-Authenticate: NTLM/Negotiate` — ретрай через
`ntlmssp.Negotiator` (см. docs/EWS-REFERENCE.md §2).

## Пространства имён

```text
soap = http://schemas.xmlsoap.org/soap/envelope/
t    = http://schemas.microsoft.com/exchange/services/2006/types
m    = http://schemas.microsoft.com/exchange/services/2006/messages
```

## Скелеты (тело soap:Body)

FindFolder:
```xml
<m:FindFolder Traversal="Shallow">
  <m:FolderShape><t:BaseShape>Default</t:BaseShape></m:FolderShape>
  <m:ParentFolderIds><t:DistinguishedFolderId Id="msgfolderroot"/></m:ParentFolderIds>
</m:FindFolder>
```

FindItem (пагинация):
```xml
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>IdOnly</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:Subject"/>
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
      <t:FieldURI FieldURI="item:IsRead"/>
      <t:FieldURI FieldURI="item:HasAttachments"/>
      <t:FieldURI FieldURI="message:From"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:IndexedPageItemView MaxEntriesReturned="50" Offset="0" BasePoint="Beginning"/>
  <m:SortOrder><t:FieldOrder Order="Descending">
    <t:FieldURI FieldURI="item:DateTimeReceived"/>
  </t:FieldOrder></m:SortOrder>
  <m:ParentFolderIds><t:DistinguishedFolderId Id="inbox"/></m:ParentFolderIds>
</m:FindItem>
```

GetItem (чтение):
```xml
<m:GetItem>
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:BodyType BodyType="Best"/>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:TextBody"/>
      <t:FieldURI FieldURI="message:ToRecipients"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:ItemIds><t:ItemId Id="..." ChangeKey="..."/></m:ItemIds>
</m:GetItem>
```

GetAttachment (скачивание):
```xml
<m:GetAttachment>
  <m:AttachmentShape><t:BaseShape>IdOnly</t:BaseShape></m:AttachmentShape>
  <m:AttachmentIds><t:AttachmentId Id="..."/></m:AttachmentIds>
</m:GetAttachment>
```

CalendarView (встречи; БЕЗ IndexedPageItemView, BaseShape=Default):
```xml
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="calendar:Location"/>
      <t:FieldURI FieldURI="calendar:Organizer"/>
      <t:FieldURI FieldURI="calendar:MyResponseType"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:CalendarView StartDate="2026-11-01T00:00:00Z" EndDate="2026-11-30T23:59:59Z"
                  MaxEntriesReturned="500"/>
  <m:SortOrder><t:FieldOrder Order="Ascending">
    <t:FieldURI FieldURI="calendar:Start"/>
  </t:FieldOrder></m:SortOrder>
  <m:ParentFolderIds><t:DistinguishedFolderId Id="calendar"/></m:ParentFolderIds>
</m:FindItem>
```

## Ключевые факты

- Пагинация: следующий `Offset` = `IndexedPagingOffset`; стоп при
  `IncludesLastItemInRange="true"`.
- CalendarView сам раскрывает повторяющиеся встречи; `MyResponseType` — ваш
  статус ответа (Accept/Tentative/Decline/NoResponseReceived).
- Вложения приходят в GetItem только метаданными (AttachmentId, Name,
  ContentType, ContentLength); байты — из GetAttachment (`t:Content`, base64).
- Ошибки: `ResponseClass="Error"` + `ResponseCode` (ErrorAccessDenied,
  ErrorItemNotFound, …); конвертные — `soap:Fault`.
- Разрешённые DistinguishedFolderId: inbox, sentitems, drafts, deleteditems,
  outbox, junkemail, calendar, contacts, tasks, msgfolderroot, root.

После генерации XML-кода — прогнать шаблоны через `xmllint --noout`.
