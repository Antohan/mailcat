package mail

import (
	"strings"

	"github.com/k3a/html2text"
)

// HTMLToText конвертирует HTML-тело письма в текст ОФЛАЙН: html2text не ходит
// в сеть (remote-контент не загружается — docs/SECURITY.md §4.5; сетевого
// доступа из пакета вообще нет — тест-сторож в mail_test.go).
func HTMLToText(html string) string {
	if html == "" {
		return ""
	}
	return strings.TrimSpace(html2text.HTML2Text(html))
}
