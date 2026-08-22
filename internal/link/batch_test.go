package link

import (
	"context"
	"testing"
)

func TestBulkCreateDuplicateCodeRollsBack(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	// 先落库一条占用 shortcode "dup" 的链接。
	if _, err := svc.Create(ctx, CreateReq{TargetURL: "https://x.com", CustomCode: "dup"}); err != nil {
		t.Fatal(err)
	}

	// 批量创建三项：前两项合法且不冲突，第三项的 CustomCode 与已存在记录冲突。
	// 整批应当失败，且前两项绝不应残留。
	_, err := svc.BulkCreate(ctx, []CreateReq{
		{TargetURL: "https://a.com", CustomCode: "a1"},
		{TargetURL: "https://b.com", CustomCode: "b1"},
		{TargetURL: "https://c.com", CustomCode: "dup"},
	})
	if err == nil {
		t.Fatal("expected batch to fail on duplicate code")
	}

	// 失败后数据库里不应出现 a1 / b1——整批回滚，数据保持完整。
	for _, code := range []string{"a1", "b1"} {
		got, gerr := svc.store.GetLinkByCode(ctx, code)
		if gerr != nil {
			t.Fatalf("get %q: %v", code, gerr)
		}
		if got.Code != "" {
			t.Fatalf("code %q should not have been persisted after batch failure", code)
		}
	}

	// 原已存在的 "dup" 仍应完好。
	dup, gerr := svc.store.GetLinkByCode(ctx, "dup")
	if gerr != nil {
		t.Fatalf("get dup: %v", gerr)
	}
	if dup.Code != "dup" {
		t.Fatal("pre-existing dup link should remain intact")
	}
}

func TestBulkCreateIntraBatchDuplicateRollsBack(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	// 同一批内两项使用相同 CustomCode，应在写库前被拦截，整批不落库。
	_, err := svc.BulkCreate(ctx, []CreateReq{
		{TargetURL: "https://a.com", CustomCode: "same"},
		{TargetURL: "https://b.com", CustomCode: "same"},
	})
	if err == nil {
		t.Fatal("expected error for duplicated custom code within batch")
	}
	got, gerr := svc.store.GetLinkByCode(ctx, "same")
	if gerr != nil {
		t.Fatalf("get same: %v", gerr)
	}
	if got.Code != "" {
		t.Fatal("no links should be persisted when batch fails")
	}
}

func TestBulkCreateAutoCodesAllPersisted(t *testing.T) {
	svc := newSvc(t)
	ctx := context.Background()

	created, err := svc.BulkCreate(ctx, []CreateReq{
		{TargetURL: "https://a.com"},
		{TargetURL: "https://b.com"},
		{TargetURL: "https://c.com"},
	})
	if err != nil {
		t.Fatalf("bulk create: %v", err)
	}
	if len(created) != 3 {
		t.Fatalf("created = %d, want 3", len(created))
	}
	seen := map[string]bool{}
	for _, l := range created {
		seen[l.Code] = true
		if l.ID == 0 {
			t.Fatal("link id not assigned")
		}
		got, gerr := svc.store.GetLinkByCode(ctx, l.Code)
		if gerr != nil {
			t.Fatalf("get %q: %v", l.Code, gerr)
		}
		if got.Code != l.Code {
			t.Fatalf("link %q not persisted", l.Code)
		}
	}
	if len(seen) != 3 {
		t.Fatalf("auto-generated codes not unique within batch: %v", created)
	}
}
