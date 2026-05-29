package store

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestInsertAndAggregate(t *testing.T) {
	s := newTestStore(t)

	// 4 active Cursor/blog, 2 active Chrome/sns, 1 idle.
	events := []Event{
		{TS: 100, App: "Cursor", Title: "a", Project: "blog", Category: "work"},
		{TS: 105, App: "Cursor", Title: "a", Project: "blog", Category: "work"},
		{TS: 110, App: "Cursor", Title: "a", Project: "blog", Category: "work"},
		{TS: 115, App: "Cursor", Title: "a", Project: "blog", Category: "work"},
		{TS: 120, App: "Google Chrome", Title: "x", Project: "sns", Category: "sns"},
		{TS: 125, App: "Google Chrome", Title: "x", Project: "sns", Category: "sns"},
		{TS: 130, App: "Cursor", Title: "a", IsIdle: true, Project: "blog", Category: "work"},
	}
	for _, e := range events {
		if err := s.Insert(e); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	const interval = 5
	active, err := s.ActiveSeconds(0, 1000, interval)
	if err != nil {
		t.Fatalf("ActiveSeconds: %v", err)
	}
	if active != 6*interval {
		t.Errorf("ActiveSeconds = %d, want %d", active, 6*interval)
	}

	idle, err := s.IdleSeconds(0, 1000, interval)
	if err != nil {
		t.Fatalf("IdleSeconds: %v", err)
	}
	if idle != 1*interval {
		t.Errorf("IdleSeconds = %d, want %d", idle, interval)
	}

	byApp, err := s.ByApp(0, 1000, interval)
	if err != nil {
		t.Fatalf("ByApp: %v", err)
	}
	if len(byApp) != 2 || byApp[0].Name != "Cursor" || byApp[0].Seconds != 4*interval {
		t.Errorf("ByApp = %+v, want Cursor first with %d sec", byApp, 4*interval)
	}

	byProj, err := s.ByProject(0, 1000, interval)
	if err != nil {
		t.Fatalf("ByProject: %v", err)
	}
	if byProj[0].Name != "blog" || byProj[1].Name != "sns" {
		t.Errorf("ByProject order = %+v", byProj)
	}
}

func TestEventsRangeIsHalfOpen(t *testing.T) {
	s := newTestStore(t)
	for _, ts := range []int64{100, 200, 300} {
		if err := s.Insert(Event{TS: ts, App: "A", Title: ""}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.Events(100, 300) // should include 100, 200 but not 300
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].TS != 100 || got[1].TS != 200 {
		t.Errorf("Events = %+v, want ts 100,200", got)
	}
}

func TestWipe(t *testing.T) {
	s := newTestStore(t)
	for _, ts := range []int64{100, 200, 300} {
		if err := s.Insert(Event{TS: ts, App: "A", Title: ""}); err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.Wipe(250)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("Wipe deleted %d, want 2", n)
	}
	n, err = s.WipeAll()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("WipeAll deleted %d, want 1", n)
	}
}
