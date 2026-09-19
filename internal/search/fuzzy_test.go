package search

import "testing"

func TestMatch(t *testing.T) {
	ok, _ := Match("production-server", "prod")
	if !ok {
		t.Fatal("expected fuzzy match")
	}
	if ok, _ := Match("database", "xyz"); ok {
		t.Fatal("unexpected fuzzy match")
	}
}
func TestRank(t *testing.T) {
	got := Rank([]string{"alpha", "production", "prod"}, "prod", func(s string) string { return s })
	if len(got) != 2 || got[0].Value != "prod" {
		t.Fatalf("unexpected ranking: %#v", got)
	}
}
