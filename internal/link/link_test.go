package link

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"taskt111-shortlink/internal/store"
)

func newSvc(t *testing.T) *Service {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "l.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return New(s)
}

func TestCreateResolve(t *testing.T) {
	svc := newSvc(t)
	l, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://x.com"})
	if err != nil {
		t.Fatal(err)
	}
	if l.Code == "" {
		t.Fatal("empty code")
	}
	got, err := svc.Resolve(context.Background(), l.Code)
	if err != nil {
		t.Fatal(err)
	}
	if got.TargetURL != "https://x.com" {
		t.Fatal("resolve target mismatch")
	}
}

func TestResolveNotFound(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.Resolve(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestCreateInvalidURL(t *testing.T) {
	svc := newSvc(t)
	_, err := svc.Create(context.Background(), CreateReq{TargetURL: "not-a-url"})
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("want ErrInvalidURL, got %v", err)
	}
}

func TestCustomCodeUnique(t *testing.T) {
	svc := newSvc(t)
	if _, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://x.com", CustomCode: "my"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://y.com", CustomCode: "my"})
	if err == nil {
		t.Fatal("expected duplicate code error")
	}
}

func TestUpdateDelete(t *testing.T) {
	svc := newSvc(t)
	l, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://x.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Update(context.Background(), l.Code, "https://y.com", "desc", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.Delete(context.Background(), l.Code); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resolve(context.Background(), l.Code); !errors.Is(err, ErrNotFound) {
		t.Fatal("link should be gone")
	}
}

func TestStatusNoLimit(t *testing.T) {
	svc := newSvc(t)
	l, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://x.com"})
	if err != nil {
		t.Fatal(err)
	}
	st, err := svc.Status(context.Background(), l.Code)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Active {
		t.Fatal("should be active")
	}
	if st.Remaining != -1 {
		t.Fatalf("remaining = %d, want -1", st.Remaining)
	}
}
