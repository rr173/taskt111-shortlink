package click

import ("context"; "path/filepath"; "testing"; "taskt111-shortlink/internal/store")

func TestRecordRejectsUnknownLink(t *testing.T) {
    s, err := store.Open(filepath.Join(t.TempDir(), "orphan.db")); if err != nil { t.Fatal(err) }; defer s.Close()
    if _, err := New(s).Record(context.Background(), "missing", "", "", "", "fp"); err == nil { t.Fatal("unknown link click must fail") }
    report, err := s.Consistency(context.Background()); if err != nil { t.Fatal(err) }
    if report.OrphanClicks != 0 || !report.Healthy { t.Fatalf("consistency = %+v", report) }
}
