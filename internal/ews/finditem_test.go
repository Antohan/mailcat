package ews

// Тесты FindItem (docs/EWS-REFERENCE.md §5): golden-запросы (distinguished и
// FolderId), разбор страниц ответа, курсор пагинации NextOffset.

import (
	"encoding/xml"
	"testing"
	"time"
)

// TestFindItemRequestGolden: запрос по distinguished-папке inbox структурно
// совпадает с эталоном testdata/finditem_request.xml.golden (шаблон §5).
func TestFindItemRequestGolden(t *testing.T) {
	req, err := newFindMessagesRequest(parentFolderIDsRequest{
		DistinguishedFolderID: &distinguishedFolderIDRequest{ID: "inbox"},
	}, 50, 0)
	if err != nil {
		t.Fatalf("сборка запроса FindItem: %v", err)
	}
	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса FindItem: %v", err)
	}
	assertXMLLike(t, "запрос FindItem (distinguished inbox)", data, fixture(t, "finditem_request.xml.golden"))
}

// TestFindItemRequestFolderID: вариант с t:FolderId (обычная папка из
// FindFolder) — вместо DistinguishedFolderId.
func TestFindItemRequestFolderID(t *testing.T) {
	req, err := newFindMessagesRequest(parentFolderIDsRequest{
		FolderID: &folderIDRequest{ID: "AQApA0000001", ChangeKey: "AQAAAB000001"},
	}, 25, 25)
	if err != nil {
		t.Fatalf("сборка запроса FindItem: %v", err)
	}
	data, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса FindItem: %v", err)
	}
	want := []byte(`<m:FindItem
    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
    Traversal="Shallow">
  <m:ItemShape>
    <t:BaseShape>Default</t:BaseShape>
    <t:AdditionalProperties>
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
    </t:AdditionalProperties>
  </m:ItemShape>
  <m:IndexedPageItemView MaxEntriesReturned="25" Offset="25" BasePoint="Beginning"/>
  <m:SortOrder>
    <t:FieldOrder Order="Descending">
      <t:FieldURI FieldURI="item:DateTimeReceived"/>
    </t:FieldOrder>
  </m:SortOrder>
  <m:ParentFolderIds>
    <t:FolderId Id="AQApA0000001" ChangeKey="AQAAAB000001"/>
  </m:ParentFolderIds>
</m:FindItem>`)
	assertXMLLike(t, "запрос FindItem (FolderId)", data, want)
}

// TestFindMessagesBadParams: некорректные pageSize/offset отклоняются до
// отправки запроса.
func TestFindMessagesBadParams(t *testing.T) {
	for _, tc := range []struct{ pageSize, offset int }{
		{0, 0}, {-1, 0}, {50, -1},
	} {
		if _, err := newFindMessagesRequest(parentFolderIDsRequest{}, tc.pageSize, tc.offset); err == nil {
			t.Errorf("pageSize=%d offset=%d: жду ошибку, получила nil", tc.pageSize, tc.offset)
		}
	}
}

// TestFindMessagesParseP1: первая страница — курсор ведёт дальше.
func TestFindMessagesParseP1(t *testing.T) {
	var resp findItemResponse
	if err := parseResponse(fixture(t, "finditem_response_p1.xml"), &resp); err != nil {
		t.Fatalf("разбор ответа FindItem p1: %v", err)
	}
	page, err := resp.result()
	if err != nil {
		t.Fatalf("result(): %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("писем %d, хочу 3", len(page.Items))
	}
	if page.NextOffset != 3 || page.IncludesLast || page.TotalInView != 6 {
		t.Errorf("курсор страницы: %+v, хочу NextOffset=3 IncludesLast=false TotalInView=6", page)
	}
	first := page.Items[0]
	want := MessageRef{
		ItemID:         "AAMkAGI2MDE=",
		ChangeKey:      "CQAAABYA0AA=",
		Subject:        "Отчёт Q3",
		Received:       time.Date(2026, 11, 17, 10, 12, 44, 0, time.UTC),
		IsRead:         false,
		HasAttachments: true,
		FromName:       "Иванов И.",
		FromEmail:      "ivanov@company.example",
	}
	if first != want {
		t.Errorf("письмо[0] = %+v, хочу %+v", first, want)
	}
}

// TestFindMessagesParseP2: последняя страница — IncludesLastItemInRange=true.
func TestFindMessagesParseP2(t *testing.T) {
	var resp findItemResponse
	if err := parseResponse(fixture(t, "finditem_response_p2.xml"), &resp); err != nil {
		t.Fatalf("разбор ответа FindItem p2: %v", err)
	}
	page, err := resp.result()
	if err != nil {
		t.Fatalf("result(): %v", err)
	}
	if len(page.Items) != 3 {
		t.Fatalf("писем %d, хочу 3", len(page.Items))
	}
	if !page.IncludesLast {
		t.Errorf("IncludesLast = false, хочу true (последняя страница)")
	}
	if page.NextOffset != 6 || page.TotalInView != 6 {
		t.Errorf("курсор: NextOffset=%d TotalInView=%d, хочу 6/6", page.NextOffset, page.TotalInView)
	}
	if page.Items[0].Subject != "Обновление регламента" {
		t.Errorf("письмо[0].Subject = %q", page.Items[0].Subject)
	}
}

// TestNextOffset: чистый курсор пагинации (EWS-REFERENCE.md §5): следующий
// Offset = IndexedPagingOffset, стоп при IncludesLastItemInRange=true.
func TestNextOffset(t *testing.T) {
	cases := []struct {
		name       string
		page       Page
		wantOffset int
		wantMore   bool
	}{
		{"первая страница", Page{NextOffset: 3, IncludesLast: false, TotalInView: 6}, 3, true},
		{"середина", Page{NextOffset: 50, IncludesLast: false, TotalInView: 1234}, 50, true},
		{"последняя страница", Page{NextOffset: 6, IncludesLast: true, TotalInView: 6}, 0, false},
		{"пустая папка", Page{NextOffset: 0, IncludesLast: true, TotalInView: 0}, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			offset, more := NextOffset(tc.page)
			if offset != tc.wantOffset || more != tc.wantMore {
				t.Errorf("NextOffset(%+v) = (%d, %v), хочу (%d, %v)",
					tc.page, offset, more, tc.wantOffset, tc.wantMore)
			}
		})
	}
}
