package terminal

import "testing"

func TestBuffer(t *testing.T) {
	b := NewBuffer(100)
	b.Append("one\ntwo\nthree")
	hits := b.Search("two")
	if len(hits) != 1 || hits[0] != 1 {
		t.Fatalf("unexpected hits: %#v", hits)
	}
	if len(b.Lines()) != 3 {
		t.Fatal("unexpected line count")
	}
}
func TestClampSize(t *testing.T) {
	s := ClampSize(1, 1)
	if s.Width != 20 || s.Height != 5 {
		t.Fatalf("unexpected size: %#v", s)
	}
}
