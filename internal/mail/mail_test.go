package mail

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// Тест-сторож сети (TESTING.md §2): пакет internal/mail не должен ходить в сеть.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("исходящий HTTP-запрос из internal/mail")
}

func TestMain(m *testing.M) {
	http.DefaultTransport = failingTransport{}
	os.Exit(m.Run())
}

// TestParseAddress: разбор "Имя <addr>" и голого адреса; мусор — ошибка.
func TestParseAddress(t *testing.T) {
	cases := []struct {
		in    string
		name  string
		email string
		ok    bool
	}{
		{"Иванов И. <ivanov@company.example>", "Иванов И.", "ivanov@company.example", true},
		{"ivanov@company.example", "", "ivanov@company.example", true},
		{"<ivanov@company.example>", "", "ivanov@company.example", true},
		{"", "", "", false},
		{"это не адрес", "", "", false},
	}
	for _, tc := range cases {
		got, err := ParseAddress(tc.in)
		if !tc.ok {
			if err == nil {
				t.Errorf("ParseAddress(%q) = %v, хочу ошибку", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseAddress(%q): %v", tc.in, err)
			continue
		}
		if got.Name != tc.name || got.Email != tc.email {
			t.Errorf("ParseAddress(%q) = {%q %q}, хочу {%q %q}", tc.in, got.Name, got.Email, tc.name, tc.email)
		}
	}
}

// TestAddressString: Address.String() даёт "Имя <email>" или голый адрес.
func TestAddressString(t *testing.T) {
	if got := (Address{Name: "Иванов И.", Email: "ivanov@company.example"}).String(); got != "Иванов И. <ivanov@company.example>" {
		t.Errorf("String() = %q", got)
	}
	if got := (Address{Email: "ivanov@company.example"}).String(); got != "ivanov@company.example" {
		t.Errorf("String() = %q", got)
	}
}

// TestResponseTypeLabel: статус ответа участника — на русском.
func TestResponseTypeLabel(t *testing.T) {
	cases := map[string]string{
		"Accept":             "принято",
		"Tentative":          "под вопросом",
		"Decline":            "отклонено",
		"NoResponseReceived": "нет ответа",
		"Organizer":          "организатор",
		"Unknown":            "неизвестно",
		"Что-то странное":    "неизвестно",
	}
	for in, want := range cases {
		if got := ResponseTypeLabel(in); got != want {
			t.Errorf("ResponseTypeLabel(%q) = %q, хочу %q", in, got, want)
		}
	}
}

// TestAttachmentSizeLabel: человекочитаемый размер вложения.
func TestAttachmentSizeLabel(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 Б"},
		{512, "512 Б"},
		{2048, "2.0 КБ"},
		{1536, "1.5 КБ"},
		{2 * 1024 * 1024, "2.0 МБ"},
		{1024 * 1024 * 1024, "1.0 ГБ"},
	}
	for _, tc := range cases {
		a := Attachment{ContentLength: tc.bytes}
		if got := a.SizeLabel(); got != tc.want {
			t.Errorf("SizeLabel(%d) = %q, хочу %q", tc.bytes, got, tc.want)
		}
	}
}

// TestEventDuration: длительность встречи считается из Start/End.
func TestEventDuration(t *testing.T) {
	ev := Event{
		Start: time.Date(2026, 11, 17, 10, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 11, 17, 11, 30, 0, 0, time.UTC),
	}
	if got := ev.Duration(); got != 90*time.Minute {
		t.Errorf("Duration() = %v, хочу 90m", got)
	}
}
