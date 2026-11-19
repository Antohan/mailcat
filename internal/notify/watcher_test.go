package notify

// Тесты watcher'а (PLAN.md этап 6): детектор новых писем по верхнему ItemId
// (первый прогон — без уведомлений), экспоненциальный backoff до 5 минут.

import (
	"testing"
	"time"

	"mailcat/internal/mail"
)

func item(id string) mail.MessageSummary { return mail.MessageSummary{ItemID: id} }

func TestNewSince(t *testing.T) {
	page := []mail.MessageSummary{item("X"), item("A"), item("B")}
	if got := NewSince("", page); got != nil {
		t.Errorf("первый прогон: получены «новые» %v — состояние только запоминается", got)
	}
	got := NewSince("A", page)
	if len(got) != 1 || got[0].ItemID != "X" {
		t.Errorf("NewSince(A) = %v, хочу [X]", got)
	}
	if got := NewSince("X", page); len(got) != 0 {
		t.Errorf("без изменений: %v", got)
	}
	// lastTop выпал из страницы (много новых) — вся страница новые.
	got = NewSince("Z", page)
	if len(got) != 3 {
		t.Errorf("lastTop за страницей: %v", got)
	}
}

func TestNextBackoff(t *testing.T) {
	base := 30 * time.Second
	cases := []struct {
		fails int
		want  time.Duration
	}{{0, 30 * time.Second}, {1, time.Minute}, {2, 2 * time.Minute}, {3, 4 * time.Minute}, {4, 5 * time.Minute}, {10, 5 * time.Minute}}
	for _, tc := range cases {
		if got := NextBackoff(base, tc.fails); got != tc.want {
			t.Errorf("NextBackoff(%v, %d) = %v, хочу %v", base, tc.fails, got, tc.want)
		}
	}
}
