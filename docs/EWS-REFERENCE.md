# EWS: точный справочник для mailcat

> **Правило №1:** никогда не придумывай элементы EWS. Используй только шаблоны и
> FieldURI из этого файла. Если чего-то здесь нет — сначала спроси пользователя,
> приложив ссылку на официальную документацию (раздел «Источники»).

## 1. Транспорт

- **Endpoint:** `POST https://<server>/EWS/Exchange.asmx`
- **Content-Type:** `text/xml; charset=utf-8`
- **Аутентификация:** Basic (`Authorization: Basic base64(user:pass)`). Если сервер
  отвечает `401` с заголовком `WWW-Authenticate: NTLM` или `Negotiate` — повторить
  запрос через NTLM-транспорт (см. §2).
- **Кодировки:** запрос и ответ — UTF-8.

## 2. NTLM в Go

```go
import "github.com/Azure/go-ntlmssp"

base := &http.Transport{TLSClientConfig: tlsCfg /* изолированный пул, см. SECURITY.md §1 */}
client := &http.Client{Transport: &ntlmssp.Negotiator{RoundTripper: base}}
// На запрос всё равно ставится SetBasicAuth(user, pass):
// Negotiator перехватывает 401-й ответ с NTLM-челленджем и доигрывает handshake.
req.SetBasicAuth(user, pass)
```

Логин может требовать домен: `COMPANY\\ivanov` или UPN `ivanov@company.example` —
обо варианта должно хватить; выбрать по поведению сервера.

## 3. Пространства имён SOAP

```text
soap  = http://schemas.xmlsoap.org/soap/envelope/           (префикс soap:)
t     = http://schemas.microsoft.com/exchange/services/2006/types      (префикс t:)
m     = http://schemas.microsoft.com/exchange/services/2006/messages   (префикс m:)
```

Общий каркас запроса (все шаблоны ниже — тело `<soap:Body>`):

```xml
<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope
    xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <!-- операция -->
  </soap:Body>
</soap:Envelope>
```

Необязательный SOAP-заголовок (использовать только при необходимости):
`<soap:Header><t:RequestServerVersion Version="Exchange2016"/></soap:Header>` —
если сервер отклоняет запросы без версии, начни с `Exchange2013_SP1`.

## 4. FindFolder — список папок

```xml
<m:FindFolder Traversal="Shallow">
  <m:FolderShape>
    <t:BaseShape>Default</t:BaseShape>
  </m:FolderShape>
  <m:ParentFolderIds>
    <t:DistinguishedFolderId Id="msgfolderroot"/>
  </m:ParentFolderIds>
</m:FindFolder>
```

Ответ (сокращённо):

```xml
<m:FindFolderResponse>
  <m:ResponseMessages>
    <m:FindFolderResponseMessage ResponseClass="Success">
      <m:ResponseCode>NoError</m:ResponseCode>
      <m:RootFolder TotalNumberOfFoldersInView="4">
        <t:Folders>
          <t:Folder>
            <t:FolderId Id="AQApA=" ChangeKey="AQAAAB"/>
            <t:DisplayName>Входящие</t:DisplayName>
            <t:TotalCount>1234</t:TotalCount>
            <t:UnreadCount>23</t:UnreadCount>
            <t:ChildFolderCount>2</t:ChildFolderCount>
          </t:Folder>
          <!-- ... -->
        </t:Folders>
      </m:RootFolder>
    </m:FindFolderResponseMessage>
  </m:ResponseMessages>
</m:FindFolderResponse>
```

Замечания:
- `Traversal`: `Shallow` | `Deep` | `SoftDeleted`. Для дерева папок делаем `Shallow`
  от `msgfolderroot` и рекурсивно `Shallow` по ChildFolderCount > 0 (Deep возвращает всё,
  но ограничен ~10 000 элементов — для MVP хватит двух уровней глубины).
- `BaseShape: Default` уже включает `DisplayName`, `TotalCount`, `UnreadCount`,
  `ChildFolderCount` — запрашивать их через AdditionalProperties не нужно.

## 5. FindItem — список писем с пагинацией

```xml
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:IndexedPageItemView MaxEntriesReturned="50" Offset="0" BasePoint="Beginning"/>
  <m:SortOrder>
    <t:FieldOrder Order="Descending">
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
    </t:FieldOrder>
  </m:SortOrder>
  <m:ParentFolderIds>
    <t:DistinguishedFolderId Id="inbox"/>
    <!-- или: <t:FolderId Id="..."/> для обычной папки из FindFolder -->
  </m:ParentFolderIds>
</m:FindItem>
```

Форма `Default` (не `IdOnly` + список FieldURI): Default сам возвращает
`Subject`, `IsRead`, `From`, `HasAttachments`; `DateTimeReceived` в Default
нет — добирается единственным FieldURI. Проверено на живом Exchange 2019
(15.2.2562): `item:IsRead` в AdditionalProperties → HTTP 500
`ErrorInvalidRequest` — не использовать (остальные FieldURI проходят).

Ответ (сокращённо):

```xml
<m:FindItemResponse>
  <m:ResponseMessages>
    <m:FindItemResponseMessage ResponseClass="Success">
      <m:ResponseCode>NoError</m:ResponseCode>
      <m:RootFolder IndexedPagingOffset="50" TotalItemsInView="1234" IncludesLastItemInRange="false">
        <t:Items>
          <t:Message>
            <t:ItemId Id="AAMkAGI..." ChangeKey="CQAAAB"/>
            <t:Subject>Отчёт Q3</t:Subject>
            <t:DateTimeReceived>2026-11-17T10:12:44Z</t:DateTimeReceived>
            <t:IsRead>false</t:IsRead>
            <t:HasAttachments>true</t:HasAttachments>
            <t:From><t:Mailbox><t:Name>Иванов И.</t:Name>
              <t:EmailAddress>ivanov@company.example</t:EmailAddress></t:Mailbox></t:From>
          </t:Message>
          <!-- ... -->
        </t:Items>
      </m:RootFolder>
    </m:FindItemResponseMessage>
  </m:ResponseMessages>
</m:FindItemResponse>
```

Пагинация: следующий запрос с `Offset = IndexedPagingOffset`; стоп, когда
`IncludesLastItemInRange = true`.

## 6. GetItem — чтение одного письма

```xml
<m:GetItem>
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
    <t:ItemId Id="AAMkAGI..." ChangeKey="CQAAAB"/>
  </m:ItemIds>
</m:GetItem>
```

Ответ — `m:GetItemResponseMessage/m:Items/t:Message` с:
- `<t:Body BodyType="Html">…</t:Body>` (или `Text`) — при `BodyType="Best"` сервер сам
  выбирает лучший вариант; HTML конвертируем в текст на клиенте (html2text);
- `<t:TextBody>` — готовый текстовый вариант (если сервер вернул);
- `<t:Attachments>` — список вложений:
  `<t:FileAttachment><t:AttachmentId Id="AAMk..."/><t:Name>report.pdf</t:Name>
  <t:ContentType>application/pdf</t:ContentType><t:ContentLength>2097152</t:ContentLength>
  </t:FileAttachment>`. В GetItem содержимое НЕ приходит — только метаданные.

## 7. GetAttachment — скачивание вложения

```xml
<m:GetAttachment>
  <m:AttachmentShape>
    <t:BaseShape>IdOnly</t:BaseShape>
  </m:AttachmentShape>
  <m:AttachmentIds>
    <t:AttachmentId Id="AAMkAGI..."/>
  </m:AttachmentIds>
</m:GetAttachment>
```

Ответ: `m:GetAttachmentResponseMessage/m:Attachments/t:FileAttachment` с
`<t:Content>` — base64 файла. Перед скачиванием проверить `ContentLength` против
лимита `max_attachment_mb`.

## 8. FindItem + CalendarView — встречи за период

```xml
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="calendar:Location"/>
      <t:FieldURI FieldURI="calendar:Organizer"/>
      <t:FieldURI FieldURI="calendar:MyResponseType"/>
      <t:FieldURI FieldURI="calendar:RequiredAttendees"/>
      <t:FieldURI FieldURI="calendar:OptionalAttendees"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:CalendarView StartDate="2026-11-01T00:00:00Z" EndDate="2026-11-30T23:59:59Z"
                  MaxEntriesReturned="500"/>
  <m:SortOrder>
    <t:FieldOrder Order="Ascending">
      <t:FieldURI FieldURI="calendar:Start"/>
    </t:FieldOrder>
  </m:SortOrder>
  <m:ParentFolderIds>
    <t:DistinguishedFolderId Id="calendar"/>
  </m:ParentFolderIds>
</m:FindItem>
```

Ответ — элементы `t:CalendarItem`:

```xml
<t:CalendarItem>
  <t:ItemId Id="AAMkAGI..." ChangeKey="DwAAAB"/>
  <t:Subject>Синк команды</t:Subject>
  <t:Start>2026-11-17T10:00:00Z</t:Start>
  <t:End>2026-11-17T10:30:00Z</t:End>
  <t:Location>Zoom</t:Location>
  <t:CalendarItemType>Occurrence</t:CalendarItemType>
  <t:Organizer><t:Mailbox><t:Name>Иванов И.</t:Name>
    <t:EmailAddress>ivanov@company.example</t:EmailAddress></t:Mailbox></t:Organizer>
  <t:MyResponseType>Accept</t:MyResponseType>
</t:CalendarItem>
```

Ключевые факты:
- **CalendarView автоматически раскрывает повторяющиеся встречи** в конкретные
  экземпляры — отдельная логика recurring не нужна. `CalendarItemType` = `Single` |
  `Occurrence` | `Exception`.
- `MyResponseType` = `Unknown | Organizer | Tentative | Accept | Decline | NoResponseReceived`.
  Это read-only свойство — показываем «вы приняли/под вопросом/не ответили».
- С `CalendarView` НЕ используется `IndexedPageItemView` — пагинации нет, ставим
  разумный `MaxEntriesReturned`.
- Даты — xs:dateTime c смещением; в запросе используем UTC (`Z`), в UI показываем в
  локальной зоне.

## 9. Справочник значений

**DistinguishedFolderId** (те, что нужны нам): `inbox`, `sentitems`, `drafts`,
`deleteditems`, `outbox`, `junkemail`, `calendar`, `contacts`, `tasks`,
`msgfolderroot`, `root`.

**FieldURI, используемые в проекте** (и только они):

| FieldURI | Что это |
|---|---|
| `item:Subject`, `item:DateTimeReceived`, `item:DateTimeSent` | тема, даты |
| `item:IsRead`, `item:HasAttachments` | флаги |
| `item:TextBody`, `item:Body` (через BodyType) | тело |
| `message:From`, `message:Sender` | отправитель |
| `message:ToRecipients`, `message:CcRecipients` | получатели |
| `message:InternetMessageId` | Message-ID |
| `folder:DisplayName` | имя папки (при IdOnly) |
| `calendar:Start`, `calendar:End`, `calendar:Location` | встреча |
| `calendar:Organizer`, `calendar:MyResponseType` | встреча |
| `calendar:RequiredAttendees`, `calendar:OptionalAttendees` | участники |
| `calendar:CalendarItemType` | Single/Occurrence/Exception |

**Ошибки EWS**, которые надо различать (часть `m:*ResponseMessage ResponseClass="Error"`):
`NoError`, `ErrorAccessDenied` (проблема прав), `ErrorFolderNotFound`,
`ErrorItemNotFound` (письмо удалено/перемещено — обновить список),
`ErrorConnectionFailed` (сеть/VPN), `ErrorInternalServerError` (повторить позже).
Транспортные: HTTP 401 (аутентификация), 403 (запрещено, возможно EWS отключён
политикой для ящика), 503 (служба недоступна).

**SOAP-fault** (вместо ResponseMessage, при ошибках конверта):
`soap:Envelope/soap:Body/soap:Fault/faultcode` + `faultstring` — тоже парсить.

## 10. Источники (официальные)

- FindItem operation: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/finditem-operation
- GetItem operation: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/getitem-operation
- FindFolder operation: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/findfolder-operation
- GetAttachment operation: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/getattachment-operation
- CalendarView: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/calendarview
- Элементы EWS XML: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/ews-xml-elements-in-exchange
- Аутентификация и EWS: https://learn.microsoft.com/en-us/exchange/client-developer/exchange-web-services/authentication-and-ews-in-exchange
- FieldURI: https://learn.microsoft.com/en-us/exchange/client-developer/web-service-reference/fielduri
