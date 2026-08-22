package link

import ("context"; "errors"; "testing"; "taskt111-shortlink/internal/click")

func TestBug01_ClickLimitUsesRecordedCountAtTheBoundary(t *testing.T) {
    svc := newSvc(t); ctx := context.Background()
    l, err := svc.Create(ctx, CreateReq{TargetURL: "https://example.com/guide", MaxClicks: 1}); if err != nil { t.Fatal(err) }
    recorder := click.New(svc.store); if _, err = recorder.Record(ctx, l.Code, "https://ref.example", "ua", "127.0.0.1", "viewer"); err != nil { t.Fatal(err) }
    if _, err = svc.Resolve(ctx, l.Code); !errors.Is(err, ErrLimitReached) { t.Fatalf("resolve error=%v", err) }
}
