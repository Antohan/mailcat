---
name: ews-reference
description: Точные SOAP-шаблоны Exchange Web Services для mailcat — использовать при любой работе с EWS/XML, чтобы не выдумывать схему
---

# EWS Reference (compact)

Full reference: `docs/EWS-REFERENCE.md` (repo root). Below are the minimal
skeletons for quick orientation. Rule: an element, attribute or FieldURI that
appears neither here nor there does NOT exist — ask the user.

## Transport

`POST https://<server>/EWS/Exchange.asmx`, `Content-Type: text/xml; charset=utf-8`,
Basic auth; on 401 with `WWW-Authenticate: NTLM/Negotiate` retry through
`ntlmssp.Negotiator` (see docs/EWS-REFERENCE.md §2).

## Namespaces

```text
soap = http://schemas.xmlsoap.org/soap/envelope/
t    = http://schemas.microsoft.com/exchange/services/2006/types
m    = http://schemas.microsoft.com/exchange/services/2006/messages
```

## Skeletons (soap:Body content)

FindFolder:
```xml
<m:FindFolder Traversal="Shallow">
  <m:FolderShape><t:BaseShape>Default</t:BaseShape></m:FolderShape>
  <m:ParentFolderIds><t:DistinguishedFolderId Id="msgfolderroot"/></m:ParentFolderIds>
</m:FindFolder>
```

FindItem (paging):
```xml
<m:FindItem Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:IndexedPageItemView MaxEntriesReturned="50" Offset="0" BasePoint="Beginning"/>
  <m:SortOrder><t:FieldOrder Order="Descending">
    <t:FieldURI FieldURI="item:DateTimeReceived"/>
  </t:FieldOrder></m:SortOrder>
  <m:ParentFolderIds><t:DistinguishedFolderId Id="inbox"/></m:ParentFolderIds>
</m:FindItem>
```

GetItem (read):
```xml
<m:GetItem>
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:TextBody"/>
      <t:FieldURI FieldURI="message:ToRecipients"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:ItemIds><t:ItemId Id="..." ChangeKey="..."/></m:ItemIds>
</m:GetItem>
```

GetAttachment (download):
```xml
<m:GetAttachment>
  <m:AttachmentShape><t:BaseShape>IdOnly</t:BaseShape></m:AttachmentShape>
  <m:AttachmentIds><t:AttachmentId Id="..."/></m:AttachmentIds>
</m:GetAttachment>
```

CalendarView (meetings; NO IndexedPageItemView, SortOrder or
Restriction, BaseShape=Default):
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
  <m:ParentFolderIds><t:DistinguishedFolderId Id="calendar"/></m:ParentFolderIds>
</m:FindItem>
```

## Key facts

- Paging: the next `Offset` is `IndexedPagingOffset`; stop at
  `IncludesLastItemInRange="true"`.
- CalendarView expands recurring meetings itself; `MyResponseType` is your own
  reply status (Accept/Tentative/Decline/NoResponseReceived).
- GetItem returns attachments as metadata only (AttachmentId, Name,
  ContentType, ContentLength); the bytes come from GetAttachment (`t:Content`,
  base64).
- Errors: `ResponseClass="Error"` + `ResponseCode` (ErrorAccessDenied,
  ErrorItemNotFound, …); envelope-level — `soap:Fault`.
- Allowed DistinguishedFolderId: inbox, sentitems, drafts, deleteditems, outbox,
  junkemail, calendar, contacts, tasks, msgfolderroot, root.

After generating XML code, wrap each fragment in the envelope from
docs/EWS-REFERENCE.md §3 (with all three namespaces) and run
`xmllint --noout` on the wrapped document — a bare fragment without namespace
declarations fails with false errors.
