package click

import ("context"; "path/filepath"; "testing"; "taskt111-shortlink/internal/store")

func TestRecordEnforcesLinkClickLimit(t *testing.T) {
    s, err := store.Open(filepath.Join(t.TempDir(), "limit.db")); if err != nil { t.Fatal(err) }; defer s.Close()
    if _, err := s.InsertLink(context.Background(), store.Link{Code: "limited", TargetURL: "https://example.com", MaxClicks: 1}); err != nil { t.Fatal(err) }
    svc := New(s)
    if _, err := svc.Record(context.Background(), "limited", "", "", "", "fp"); err != nil { t.Fatal(err) }
    if _, err := svc.Record(context.Background(), "limited", "", "", "", "fp"); err == nil { t.Fatal("second click must be rejected") }
    n, err := s.CountClicks(context.Background(), "limited"); if err != nil { t.Fatal(err) }
    if n != 1 { t.Fatalf("click count = %d, want 1", n) }
}
