package ews

// Тесты FindItem+CalendarView (docs/EWS-REFERENCE.md §8): golden-запрос
// (БЕЗ IndexedPageItemView!), разбор t:CalendarItem с раскрытыми повторами.

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCalendarRequestGolden: запрос CalendarView структурно совпадает с §8:
// BaseShape Default + calendar:* поля, m:CalendarView@StartDate/@EndDate/
// MaxEntriesReturned, папка calendar. Без SortOrder: с CalendarView EWS его
// запрещает (ErrorInvalidOperation), встречи приходят по Start по возрастанию.
func TestCalendarRequestGolden(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)
	data, err := xml.MarshalIndent(newFindEventsRequest(start, end), "", "  ")
	if err != nil {
		t.Fatalf("маршалинг запроса CalendarView: %v", err)
	}
	want := []byte(`<m:FindItem
	    xmlns:m="http://schemas.microsoft.com/exchange/services/2006/messages"
	    xmlns:t="http://schemas.microsoft.com/exchange/services/2006/types"
	    Traversal="Shallow">
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
	  <m:CalendarView StartDate="2026-11-01T00:00:00Z" EndDate="2026-11-30T23:59:59Z" MaxEntriesReturned="500"/>
	  <m:ParentFolderIds>
	    <t:DistinguishedFolderId Id="calendar"/>
	  </m:ParentFolderIds>
	</m:FindItem>`)
	assertXMLLike(t, "запрос CalendarView", data, want)
}

// TestCalendarParse: фикстура — 3 встречи (Occurrence раскрыта сервером,
// Exception и Single), статусы ответов, участники.
func TestCalendarParse(t *testing.T) {
	var resp findEventsResponse
	if err := parseResponse(fixture(t, "calendar_response.xml"), &resp); err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	events, err := resp.result()
	if err != nil {
		t.Fatalf("result: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("событий %d, хочу 3", len(events))
	}

	first := events[0]
	if first.Subject != "Синк команды" {
		t.Errorf("Subject = %q", first.Subject)
	}
	wantStart := time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC)
	if !first.Start.Equal(wantStart) {
		t.Errorf("Start = %v, хочу %v", first.Start, wantStart)
	}
	if first.End.Sub(first.Start) != 30*time.Minute {
		t.Errorf("длительность = %v, хочу 30m", first.End.Sub(first.Start))
	}
	if first.Location != "Zoom" {
		t.Errorf("Location = %q", first.Location)
	}
	if first.ItemType != "Occurrence" {
		t.Errorf("ItemType = %q, хочу Occurrence (повтор раскрыт сервером)", first.ItemType)
	}
	if first.MyResponseType != "Accept" {
		t.Errorf("MyResponseType = %q, хочу Accept", first.MyResponseType)
	}
	if first.Organizer.Name != "Иванов И." || first.Organizer.Email != "ivanov@company.example" {
		t.Errorf("Organizer = %+v", first.Organizer)
	}
	if len(first.Required) != 2 || first.Required[0].Email != "apetrova@company.example" {
		t.Errorf("Required = %+v", first.Required)
	}
	if events[1].ItemType != "Single" || events[1].MyResponseType != "Tentative" {
		t.Errorf("events[1] = %+v", events[1])
	}
	if events[2].ItemType != "Exception" || events[2].MyResponseType != "NoResponseReceived" {
		t.Errorf("events[2] = %+v", events[2])
	}
	if len(events[1].Optional) != 1 {
		t.Errorf("Optional = %+v, хочу 1 участника", events[1].Optional)
	}
}

// TestFindEvents: полный цикл через mock-сервер с приватным пулом.
func TestFindEvents(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "CalendarView") {
			t.Errorf("запрос не содержит CalendarView: %s", body)
		}
		_, _ = w.Write(fixture(t, "calendar_response.xml"))
	}))
	defer srv.Close()

	client := newTestEWSClient(t, srv)
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)
	events, err := client.FindEvents(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FindEvents: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("событий %d, хочу 3", len(events))
	}
	if events[0].ItemID == "" || events[0].ChangeKey == "" {
		t.Errorf("ItemID/ChangeKey не разобраны: %+v", events[0])
	}
}
