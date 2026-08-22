package link

import ("context"; "testing")

func TestBulkCreateDoesNotLeavePartialRows(t *testing.T) {
    svc := newSvc(t)
    _, err := svc.BulkCreate(context.Background(), []CreateReq{{TargetURL: "https://example.com/one", CustomCode: "same"}, {TargetURL: "https://example.com/two", CustomCode: "same"}})
    if err == nil { t.Fatal("duplicate batch must fail") }
    rows, err := svc.List(context.Background(), "", 20, 0)
    if err != nil { t.Fatal(err) }
    if len(rows) != 0 { t.Fatalf("partial bulk rows = %d, want 0", len(rows)) }
}
