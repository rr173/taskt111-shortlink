package link

import ("context"; "testing")

func TestSearchTreatsWildcardAsLiteral(t *testing.T) {
    svc := newSvc(t); if _, err := svc.Create(context.Background(), CreateReq{TargetURL: "https://example.com/a", Description: "plain"}); err != nil { t.Fatal(err) }
    rows, err := svc.Search(context.Background(), "%", 20); if err != nil { t.Fatal(err) }
    if len(rows) != 0 { t.Fatalf("wildcard search returned %d rows, want 0", len(rows)) }
}
