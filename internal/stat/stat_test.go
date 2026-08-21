package stat

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"taskt111-shortlink/internal/store"
)

func newStat(t *testing.T) *Service {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "st.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := s.InsertLink(context.Background(), store.Link{Code: "k", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	return New(s)
}

func TestDailySingleDay(t *testing.T) {
	svc := newStat(t)
	now := time.Now().UnixMilli()
	if _, err := svc.store.InsertClick(context.Background(), store.Click{Code: "k", ClickedAt: now}); err != nil {
		t.Fatal(err)
	}
	day := time.UnixMilli(now).UTC().Format("2006-01-02")
	rows, err := svc.DailyBreakdown(context.Background(), "k", day, day)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Count != 1 {
		t.Fatalf("daily = %+v, want 1 day count 1", rows)
	}
}

func TestTopLinksNormalCtx(t *testing.T) {
	svc := newStat(t)
	now := time.Now().UnixMilli()
	if _, err := svc.store.InsertClick(context.Background(), store.Click{Code: "k", ClickedAt: now}); err != nil {
		t.Fatal(err)
	}
	top, err := svc.TopLinks(context.Background(), "", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(top) != 1 || top[0].Code != "k" {
		t.Fatalf("top = %+v, want [k]", top)
	}
}

func TestReferersAndTotal(t *testing.T) {
	svc := newStat(t)
	now := time.Now().UnixMilli()
	if _, err := svc.store.InsertClick(context.Background(), store.Click{Code: "k", ClickedAt: now, Referer: "https://r"}); err != nil {
		t.Fatal(err)
	}
	n, err := svc.TotalClicks(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("total = %d, want 1", n)
	}
	refs, err := svc.Referers(context.Background(), "k", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 1 || refs[0].Referer != "https://r" {
		t.Fatalf("refs = %+v", refs)
	}
}
