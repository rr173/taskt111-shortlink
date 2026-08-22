package click

import (
	"context"
	"errors"
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

// TestRecordRejectsUnknownCode 守护不变量：对不存在的短链记录点击必须被明确拒绝，
// 既不写入孤立点击，也不让一致性报告误报健康。
func TestRecordRejectsUnknownCode(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := New(s)

	c, recErr := svc.Record(context.Background(), "nope", "ref", "ua", "1.2.3.4", "fp")
	if !errors.Is(recErr, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", recErr)
	}
	if c != (store.Click{}) {
		t.Fatalf("no click should be returned, got %+v", c)
	}

	// 不应有任何点击落库，一致性报告应仍为健康（0 孤立）。
	rep, err := s.Consistency(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Clicks != 0 || rep.OrphanClicks != 0 {
		t.Fatalf("unknown code leaked a click: %+v", rep)
	}
	if !rep.Healthy {
		t.Fatalf("clean store should be healthy, got %+v", rep)
	}

	// 对照组：已知短链正常写入，仍保持 0 孤立且健康。
	if _, err := s.InsertLink(context.Background(), store.Link{Code: "k", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Record(context.Background(), "k", "ref", "ua", "1.2.3.4", "fp"); err != nil {
		t.Fatalf("record on existing link failed: %v", err)
	}
	rep2, _ := s.Consistency(context.Background())
	if rep2.Clicks != 1 || rep2.OrphanClicks != 0 || !rep2.Healthy {
		t.Fatalf("expected 1 click / 0 orphan / healthy, got %+v", rep2)
	}
}

