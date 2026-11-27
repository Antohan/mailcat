package notify

// Watcher — фоновый опрос «Входящих» (PLAN.md этап 6): тикер не чаще 30с,
// сравнение верхних ItemId с последним увиденным, первый прогон — запомнить
// топ без уведомлений; ошибки сети — тихий экспоненциальный backoff до 5 минут
// (уведомления об ошибках НЕ показываются), восстановление — обычный интервал.

import (
	"context"
	"time"

	"mailcat/internal/config"
	"mailcat/internal/mail"
)

// NewSince возвращает письма страницы новее lastTop (страница отсортирована
// сервером по DateTimeReceived Descending). Пустой lastTop — первый прогон:
// ничего не «новое».
func NewSince(lastTop string, page []mail.MessageSummary) []mail.MessageSummary {
	if lastTop == "" {
		return nil
	}
	for i, m := range page {
		if m.ItemID == lastTop {
			return page[:i]
		}
	}
	return page
}

// NextBackoff — пауза после fails ошибок подряд: base·2^fails, максимум 5 минут.
func NextBackoff(base time.Duration, fails int) time.Duration {
	d := base
	for i := 0; i < fails; i++ {
		d *= 2
		if d >= 5*time.Minute {
			return 5 * time.Minute
		}
	}
	return d
}

// Watcher опрашивает ящик и решает, кого уведомлять; показ и обновление UI —
// через колбэки (тестируемость без osascript/tview).
type Watcher struct {
	Client interface {
		FindDistinguishedMessages(ctx context.Context, folderID string, pageSize, offset int) (page Page, err error)
	}
	Rule       config.Notify
	OrgDomains []string
	PageSize   int
	Notify     func(title, body string) // показ (platform.Notify + sound)
	OnUnread   func(unread int)         // обновление счётчика UI

	lastTop string
	fails   int
}

// Page — минимальный контрак страницы (развязка от internal/ews).
type Page struct {
	Items []mail.MessageSummary
}

// PollSingle выполняет один шаг опроса (выделен для тестов без тикера):
// новые письма → фильтр правил → уведомления; обновляет lastTop.
func (w *Watcher) PollSingle(_ context.Context, fetch func() (Page, error), unread func() int) error {
	page, err := fetch()
	if err != nil {
		return err
	}
	w.fails = 0
	for _, m := range NewSince(w.lastTop, page.Items) {
		if !ShouldNotify(m, w.Rule, w.OrgDomains) {
			continue
		}
		title, body := NotificationTexts(m, w.Rule.ShowSubject)
		if w.Notify != nil {
			w.Notify(title, body)
		}
	}
	if len(page.Items) > 0 {
		w.lastTop = page.Items[0].ItemID
	}
	if w.OnUnread != nil && unread != nil {
		w.OnUnread(unread())
	}
	return nil
}

// Run крутит опрос до отмены ctx. interval < 30s поднимается до 30s.
// Ошибки — тихий backoff; успех — обычный интервал.
func (w *Watcher) Run(ctx context.Context, interval time.Duration, fetch func() (Page, error), unread func() int) {
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if err := w.PollSingle(ctx, fetch, unread); err != nil {
			w.fails++
			t.Reset(NextBackoff(interval, w.fails-1))
			continue
		}
		t.Reset(interval)
	}
}
