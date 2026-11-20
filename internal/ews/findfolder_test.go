package ews

// Тесты FindFolder (docs/EWS-REFERENCE.md §4): golden-запрос, разбор ответа,
// ошибка ResponseClass="Error".

import (
	"encoding/xml"
	"errors"
	"testing"
)

// TestFindFolderRequestGolden: маршалинг запроса m:FindFolder структурно
// совпадает с шаблоном EWS-REFERENCE.md §4 (Shallow, Default, msgfolderroot).
func TestFindFolderRequestGolden(t *testing.T) {
	data, err := xml.MarshalIndent(newFindFoldersRequest(), "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса FindFolder: %v", err)
	}
	want := []byte(`<m:FindFolder
    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
    Traversal="Shallow">
  <m:FolderShape>
    <t:BaseShape>Default</t:BaseShape>
  </m:FolderShape>
  <m:ParentFolderIds>
    <t:DistinguishedFolderId Id="msgfolderroot"/>
  </m:ParentFolderIds>
</m:FindFolder>`)
	assertXMLLike(t, "запрос FindFolder", data, want)
}

// TestFindFoldersParse: фикстура findfolder_response.xml → три папки
// с ожидаемыми значениями (кириллица в DisplayName).
func TestFindFoldersParse(t *testing.T) {
	var resp findFolderResponse
	if err := parseResponse(fixture(t, "findfolder_response.xml"), &resp); err != nil {
		t.Fatalf("разбор ответа FindFolder: %v", err)
	}
	folders, err := resp.result()
	if err != nil {
		t.Fatalf("result(): %v", err)
	}
	if len(folders) != 3 {
		t.Fatalf("папок %d, хочу 3", len(folders))
	}
	want := []Folder{
		{ID: "AQApA0000001", ChangeKey: "AQAAAB000001", DisplayName: "Входящие", TotalCount: 1234, UnreadCount: 23, ChildFolderCount: 2},
		{ID: "AQApA0000002", ChangeKey: "AQAAAB000002", DisplayName: "Отправленные", TotalCount: 890, UnreadCount: 0, ChildFolderCount: 1},
		{ID: "AQApA0000003", ChangeKey: "AQAAAB000003", DisplayName: "Черновики", TotalCount: 5, UnreadCount: 5, ChildFolderCount: 0},
	}
	for i, w := range want {
		if folders[i] != w {
			t.Errorf("папка[%d] = %+v, хочу %+v", i, folders[i], w)
		}
	}
}

// TestFindFoldersResponseError: ResponseClass="Error" → *EWSError с кодом
// из ответа (EWS-REFERENCE.md §9).
func TestFindFoldersResponseError(t *testing.T) {
	doc := []byte(`<?xml version="1.0" encoding="utf-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/"
               xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
               xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages">
  <soap:Body>
    <m:FindFolderResponse>
      <m:ResponseMessages>
        <m:FindFolderResponseMessage ResponseClass="Error">
          <m:MessageText>У вас нет прав на эту операцию.</m:MessageText>
          <m:ResponseCode>ErrorAccessDenied</m:ResponseCode>
        </m:FindFolderResponseMessage>
      </m:ResponseMessages>
    </m:FindFolderResponse>
  </soap:Body>
</soap:Envelope>`)
	var resp findFolderResponse
	if err := parseResponse(doc, &resp); err != nil {
		t.Fatalf("разбор ответа FindFolder: %v", err)
	}
	_, err := resp.result()
	if err == nil {
		t.Fatal("хочу EWSError")
	}
	var ewsErr *EWSError
	if !errors.As(err, &ewsErr) {
		t.Fatalf("ошибка %v (%T), хочу *EWSError", err, err)
	}
	if ewsErr.Code != "ErrorAccessDenied" {
		t.Errorf("Code = %q, хочу ErrorAccessDenied", ewsErr.Code)
	}
	if ewsErr.UserHint() == "" {
		t.Errorf("для ErrorAccessDenied должна быть подсказка")
	}
}
