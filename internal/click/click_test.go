package click

import (
	"context"
	"path/filepath"
	"testing"

	"taskt111-shortlink/internal/store"
)

func newSvc(t *testing.T) (*Service, *store.Store) {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return New(s), s
}

func TestRecordAndRecent(t *testing.T) {
	svc, s := newSvc(t)
	if _, err := s.InsertLink(context.Background(), store.Link{Code: "k", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := svc.Record(context.Background(), "k", "https://r", "ua", "1.2.3.4", "fp"); err != nil {
			t.Fatal(err)
		}
	}
	recent, err := svc.Recent(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 3 {
		t.Fatalf("recent = %d, want 3", len(recent))
	}
}

// TestRecordCountsTowardLimit verifies that each recorded click is attributed to
// the link and counted toward its MaxClicks, so the recorded count and the count
// read back agree at the boundary, and the (MaxClicks+1)-th record is rejected.
func TestRecordCountsTowardLimit(t *testing.T) {
	svc, s := newSvc(t)
	if _, err := s.InsertLink(context.Background(), store.Link{Code: "lim", TargetURL: "https://x.com", MaxClicks: 3}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		if _, err := svc.Record(context.Background(), "lim", "", "", "1.2.3.4", "fp"); err != nil {
			t.Fatalf("record #%d: %v", i, err)
		}
		n, err := s.CountClicks(context.Background(), "lim")
		if err != nil {
			t.Fatal(err)
		}
		if n != i {
			t.Fatalf("after %d records, count = %d, want %d", i, n, i)
		}
	}
	// The limit is reached; the next record must be rejected (critical count).
	if _, err := svc.Record(context.Background(), "lim", "", "", "1.2.3.4", "fp"); err == nil {
		t.Fatal("expected click limit reached error on 4th record")
	}
	n, err := s.CountClicks(context.Background(), "lim")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("count after rejected record = %d, want 3", n)
	}
}
