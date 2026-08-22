package store

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	db := filepath.Join(t.TempDir(), "t.db")
	s, err := Open(db)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestInsertGetLink(t *testing.T) {
	s := tempStore(t)
	l, err := s.InsertLink(context.Background(), Link{Code: "abc", TargetURL: "https://x.com", Owner: "o"})
	if err != nil {
		t.Fatal(err)
	}
	if l.ID == 0 {
		t.Fatal("id not set")
	}
	got, err := s.GetLinkByCode(context.Background(), "abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetURL != "https://x.com" {
		t.Fatalf("mismatch %+v", got)
	}
	miss, err := s.GetLinkByCode(context.Background(), "nope")
	if err != nil {
		t.Fatal(err)
	}
	if miss.Code != "" {
		t.Fatal("expected empty code for missing")
	}
}

func TestListCountSearch(t *testing.T) {
	s := tempStore(t)
	for i := 0; i < 3; i++ {
		_, err := s.InsertLink(context.Background(), Link{
			Code:        fmt.Sprintf("c%d", i),
			TargetURL:   "https://x.com/" + string(rune('a'+i)),
			Description: "d",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	n, err := s.CountLinks(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
	ls, err := s.ListLinks(context.Background(), "", 2, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(ls) != 2 {
		t.Fatalf("list returned %d, want 2", len(ls))
	}
	fs, err := s.SearchLinks(context.Background(), "x.com", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 3 {
		t.Fatalf("search returned %d, want 3", len(fs))
	}
}

func TestClicksDailySingleDay(t *testing.T) {
	s := tempStore(t)
	if _, err := s.InsertLink(context.Background(), Link{Code: "c1", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	if _, err := s.InsertClick(context.Background(), Click{Code: "c1", ClickedAt: now}); err != nil {
		t.Fatal(err)
	}
	n, err := s.CountClicks(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("clicks = %d, want 1", n)
	}
	day := time.UnixMilli(now).UTC().Format("2006-01-02")
	rows, err := s.DailyClicks(context.Background(), "c1", day, day)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Count != 1 {
		t.Fatalf("daily = %+v, want 1 day count 1", rows)
	}
}

func TestDeleteLinkRemovesClicks(t *testing.T) {
	s := tempStore(t)
	if _, err := s.InsertLink(context.Background(), Link{Code: "d1", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertClick(context.Background(), Click{Code: "d1", ClickedAt: time.Now().UnixMilli()}); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteLink(context.Background(), "d1"); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetLinkByCode(context.Background(), "d1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "" {
		t.Fatal("link not deleted")
	}
	n, err := s.CountClicks(context.Background(), "d1")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("clicks not deleted: %d", n)
	}
}

// TestDeleteLinkMissingReturnsNotFound 验证删除不存在的短链时返回 ErrLinkNotFound，
// 让上层据此向接口返回明确的未找到结果，而非静默成功。
func TestDeleteLinkMissingReturnsNotFound(t *testing.T) {
	s := tempStore(t)
	if err := s.DeleteLink(context.Background(), "no-such-code"); !errors.Is(err, ErrLinkNotFound) {
		t.Fatalf("want ErrLinkNotFound, got %v", err)
	}
}

func TestRestartRecovery(t *testing.T) {
	db := filepath.Join(t.TempDir(), "persist.db")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.InsertLink(context.Background(), Link{Code: "p1", TargetURL: "https://x.com"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	got, err := s2.GetLinkByCode(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Code != "p1" {
		t.Fatal("link lost after restart")
	}
}
