package state

import "testing"

func TestRecordRecent(t *testing.T) {
	s := State{RecentHostNames: []string{"b", "a"}}
	s = RecordRecent(s, "c")
	if len(s.RecentHostNames) != 3 || s.RecentHostNames[0] != "c" {
		t.Fatalf("got %#v", s.RecentHostNames)
	}
	s = RecordRecent(s, "a")
	if s.RecentHostNames[0] != "a" || len(s.RecentHostNames) != 3 {
		t.Fatalf("re-touch should move to front: %#v", s.RecentHostNames)
	}
}

func TestRemoveHost(t *testing.T) {
	s := State{RecentHostNames: []string{"a", "b", "c"}}
	s = RemoveHost(s, "b")
	want := []string{"a", "c"}
	if len(s.RecentHostNames) != len(want) {
		t.Fatalf("got %#v", s.RecentHostNames)
	}
	for i := range want {
		if s.RecentHostNames[i] != want[i] {
			t.Fatalf("got %#v", s.RecentHostNames)
		}
	}
}

func TestRenameHost(t *testing.T) {
	s := State{RecentHostNames: []string{"old", "x"}}
	s = RenameHost(s, "old", "new")
	if s.RecentHostNames[0] != "new" || s.RecentHostNames[1] != "x" {
		t.Fatalf("got %#v", s.RecentHostNames)
	}
}
