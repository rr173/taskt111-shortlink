package click_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"taskt111-shortlink/internal/click"
	"taskt111-shortlink/internal/link"
	"taskt111-shortlink/internal/store"
)

// TestRecordEnforcesMaxClicks 验证：设置了访问上限的短链，达到上限后
// 后续访问被拒绝，且数据库中的总记录不超过上限。
func TestRecordEnforcesMaxClicks(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "limit.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	lnkSvc := link.New(s)
	l, err := lnkSvc.Create(context.Background(), link.CreateReq{
		TargetURL: "https://x.com",
		MaxClicks: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	svc := click.New(s)
	// 写入上限数量的点击，应全部成功。
	for i := 0; i < 3; i++ {
		if _, err := svc.Record(context.Background(), l.Code, "r", "ua", "1.2.3.4", "fp"); err != nil {
			t.Fatalf("record %d: %v", i, err)
		}
	}
	// 达到上限后再访问，应被拒绝。
	_, err = svc.Record(context.Background(), l.Code, "r", "ua", "1.2.3.4", "fp")
	if err == nil {
		t.Fatal("expected click to be rejected after limit reached, got nil")
	}

	// 数据库中的总记录数不超过上限。
	n, err := s.CountClicks(context.Background(), l.Code)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("clicks = %d, want exactly 3 (the max)", n)
	}
	if n > l.MaxClicks {
		t.Fatalf("clicks %d exceeded max %d", n, l.MaxClicks)
	}

	// 再尝试多次，记录数仍不超过上限。
	for i := 0; i < 5; i++ {
		_, _ = svc.Record(context.Background(), l.Code, "r", "ua", "1.2.3.4", "fp")
	}
	n2, err := s.CountClicks(context.Background(), l.Code)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 3 {
		t.Fatalf("clicks after extra attempts = %d, want 3", n2)
	}
}

// TestRecordNotFound 验证记录不存在的短链时返回错误。
func TestRecordNotFound(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "nf.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	svc := click.New(s)
	_, err = svc.Record(context.Background(), "nope", "r", "ua", "1.2.3.4", "fp")
	if err == nil {
		t.Fatal("expected error for missing link")
	}
	if !errors.Is(err, click.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
