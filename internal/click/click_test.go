package click

import (
	"context"
	"path/filepath"
	"testing"

	"taskt111-shortlink/internal/store"
)

func TestRecordAndRecent(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.InsertLink(context.Background(), store.Link{Code: "k", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	svc := New(s)
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
