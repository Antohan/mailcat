package ui

import (
	"testing"

	"mailcat/internal/mail"
)

// RD-7: выделение после пересборки списка — по ItemId, при промахе — прежняя
// позиция, стянутая в пределы нового списка.
func TestRowAfterReload(t *testing.T) {
	items := []mail.MessageSummary{{ItemID: "a"}, {ItemID: "b"}, {ItemID: "c"}}
	if got := rowAfterReload(items, "b", 2); got != 1 {
		t.Fatalf("по ItemId: got %d, want 1", got)
	}
	if got := rowAfterReload(items, "", 2); got != 2 {
		t.Fatalf("без ItemId — прежняя позиция: got %d, want 2", got)
	}
	if got := rowAfterReload(items, "нет такого", 2); got != 2 {
		t.Fatalf("письмо исчезло — прежняя позиция: got %d, want 2", got)
	}
	if got := rowAfterReload(items[:1], "b", 2); got != 0 {
		t.Fatalf("список сжался — стяжка в пределы: got %d, want 0", got)
	}
}
