package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"taskt111-shortlink/internal/click"
	"taskt111-shortlink/internal/link"
	"taskt111-shortlink/internal/stat"
	"taskt111-shortlink/internal/store"
)

// runSmokeTest 执行不依赖外部服务的自检：建库、增删查、点击计数、按天聚合、
// 热门排行、批量创建与重启恢复，全部通过后返回 nil。它只走健康路径，不会触发
// 任何被注入的缺陷，因此可稳定用于容器启动校验。
func runSmokeTest() error {
	dir, err := os.MkdirTemp("", "shortlink-smoke-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	db := filepath.Join(dir, "smoke.db")

	st, err := store.Open(db)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	lnk := link.New(st)
	clk := click.New(st)
	stt := stat.New(st)

	// 1) 创建短链（自动生成短码）
	l, err := lnk.Create(context.Background(), link.CreateReq{
		TargetURL: "https://example.com/a",
		Owner:     "smoke",
	})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	if l.Code == "" {
		return errors.New("created link has empty code")
	}

	// 2) 解析（已存在、无过期、无点击上限）
	got, err := lnk.Resolve(context.Background(), l.Code)
	if err != nil {
		return fmt.Errorf("resolve: %w", err)
	}
	if got.TargetURL != "https://example.com/a" {
		return fmt.Errorf("resolve target mismatch: %s", got.TargetURL)
	}

	// 3) 记录点击并核对总量
	if _, err := clk.Record(context.Background(), l.Code, "https://ref", "ua", "1.2.3.4", "fp1"); err != nil {
		return fmt.Errorf("record: %w", err)
	}
	n, err := stt.TotalClicks(context.Background(), l.Code)
	if err != nil {
		return fmt.Errorf("total: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("expected 1 click, got %d", n)
	}

	// 4) 单日按天聚合（当天 1 次点击）：区间仅 1 天，不应丢天
	today := time.Now().UTC().Format("2006-01-02")
	days, err := stt.DailyBreakdown(context.Background(), l.Code, today, today)
	if err != nil {
		return fmt.Errorf("daily: %w", err)
	}
	if len(days) != 1 || days[0].Count != 1 {
		return fmt.Errorf("daily mismatch: %+v", days)
	}

	// 5) 热门排行（普通 ctx，不应触发取消语义）
	top, err := stt.TopLinks(context.Background(), "", 5)
	if err != nil {
		return fmt.Errorf("top: %w", err)
	}
	if len(top) != 1 || top[0].Code != l.Code {
		return fmt.Errorf("top mismatch: %+v", top)
	}

	// 6) 批量创建（唯一短码，不触发重复吞错）
	bulk, err := lnk.BulkCreate(context.Background(), []link.CreateReq{
		{TargetURL: "https://example.com/b"},
		{TargetURL: "https://example.com/c"},
	})
	if err != nil {
		return fmt.Errorf("bulk: %w", err)
	}
	if len(bulk) != 2 {
		return fmt.Errorf("bulk returned %d links", len(bulk))
	}

	// 7) 列表（直接走 store，避免分页边界）
	links, err := st.ListLinks(context.Background(), "", 2, 0)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if len(links) < 1 {
		return errors.New("list returned no links")
	}

	// 8) 重启恢复：关闭后重新打开同一文件，数据仍在
	if err := st.Close(); err != nil {
		return fmt.Errorf("close: %w", err)
	}
	st2, err := store.Open(db)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer st2.Close()
	got2, err := st2.GetLinkByCode(context.Background(), l.Code)
	if err != nil {
		return fmt.Errorf("reopen get: %w", err)
	}
	if got2.Code != l.Code {
		return errors.New("restart recovery lost link")
	}
	return nil
}
