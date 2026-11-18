package security

import (
	"net/http"
	"os"
	"testing"
)

// Тест-сторож сети (TESTING.md §2): пакет internal/security не должен ходить
// в сеть — санитайзер/анти-фишинг/имена файлов сугубо локальны.
type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	panic("исходящий HTTP-запрос из internal/security")
}

func TestMain(m *testing.M) {
	http.DefaultTransport = failingTransport{}
	os.Exit(m.Run())
}
