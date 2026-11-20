package ews

// golden_test.go — хелпер структурного сравнения XML для golden-тестов
// запросов (docs/TESTING.md §1: сравнение структурой, не сырыми строками).
//
// Канонизация: префиксы пространств имён раскрываются в URL, атрибуты
// сортируются по имени, текст тримится — две семантически одинаковые
// сериализации с разными префиксами/порядком атрибутов равны.

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// xmlTreeNode — узел канонического дерева XML.
type xmlTreeNode struct {
	Name     string // «URL локальное-имя», либо просто «локальное-имя»
	Attrs    map[string]string
	Text     string
	Children []*xmlTreeNode
}

// parseXMLTree строит каноническое дерево из XML-документа.
func parseXMLTree(t *testing.T, data []byte) *xmlTreeNode {
	t.Helper()
	root := &xmlTreeNode{Name: "#document"}
	stack := []*xmlTreeNode{root}
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("канонизация XML: %v", err)
		}
		parent := stack[len(stack)-1]
		switch tv := tok.(type) {
		case xml.StartElement:
			node := &xmlTreeNode{Name: tv.Name.Local, Attrs: map[string]string{}}
			if tv.Name.Space != "" {
				node.Name = tv.Name.Space + " " + tv.Name.Local
			}
			for _, a := range tv.Attr {
				// объявления xmlns — не данные: пространство имён уже
				// раскрыто декодером в Name.Space элемента
				if a.Name.Space == "xmlns" || (a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				name := a.Name.Local
				if a.Name.Space != "" {
					name = a.Name.Space + " " + a.Name.Local
				}
				node.Attrs[name] = a.Value
			}
			parent.Children = append(parent.Children, node)
			stack = append(stack, node)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if text := strings.TrimSpace(string(tv)); text != "" {
				parent.Text = strings.TrimSpace(parent.Text + " " + text)
			}
		}
	}
	if len(stack) != 1 {
		t.Fatal("канонизация XML: несбалансированный документ")
	}
	return root
}

// String рендерит дерево в читаемый вид для диагностики расхождений.
func (n *xmlTreeNode) String() string {
	var b strings.Builder
	n.writeTo(&b, "")
	return b.String()
}

func (n *xmlTreeNode) writeTo(b *strings.Builder, indent string) {
	if n.Name == "#document" {
		for _, ch := range n.Children {
			ch.writeTo(b, indent)
		}
		return
	}
	names := make([]string, 0, len(n.Attrs))
	for k := range n.Attrs {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Fprintf(b, "%s%s", indent, n.Name)
	for _, k := range names {
		fmt.Fprintf(b, " %s=%q", k, n.Attrs[k])
	}
	if n.Text != "" {
		fmt.Fprintf(b, " : %s", n.Text)
	}
	b.WriteByte('\n')
	for _, ch := range n.Children {
		ch.writeTo(b, indent+"  ")
	}
}

// assertXMLLike сравнивает два XML-документа структурно (канонические деревья).
func assertXMLLike(t *testing.T, context string, got, want []byte) {
	t.Helper()
	gotTree := parseXMLTree(t, got)
	wantTree := parseXMLTree(t, want)
	if !reflect.DeepEqual(gotTree, wantTree) {
		t.Errorf("%s: XML не совпадает с эталоном.\n--- получено ---\n%s--- эталон ---\n%s",
			context, gotTree, wantTree)
	}
}
