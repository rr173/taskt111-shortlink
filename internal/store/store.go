// Package store 提供短链服务的 SQLite 持久化层。
//
// 所有数据统一落库到单个 SQLite 文件，服务重启后通过重新 Open 同一文件
// 恢复全部状态（链接、点击记录、统计所需原始数据），因此具备保存与重启
// 恢复路径。小型数据统一使用嵌入式 SQLite，不依赖任何外部服务。
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Link 是短链的核心实体。
type Link struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	TargetURL   string `json:"target_url"`
	Owner       string `json:"owner"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"` // unix 毫秒
	ExpiresAt   int64  `json:"expires_at"` // unix 毫秒，0 表示不过期
	MaxClicks   int    `json:"max_clicks"` // 0 表示不限
	CustomAlias bool   `json:"custom_alias"`
}

// Click 是单次访问记录。
type Click struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	ClickedAt   int64  `json:"clicked_at"` // unix 毫秒
	Referer     string `json:"referer"`
	UserAgent   string `json:"user_agent"`
	IP          string `json:"ip"`
	Fingerprint string `json:"fingerprint"`
	Day         string `json:"day"` // YYYY-MM-DD，便于按天聚合
}

// DayStat 是某一天的点击聚合。
type DayStat struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

// RefererStat 是按来源聚合的点击。
type RefererStat struct {
	Referer string `json:"referer"`
	Count   int    `json:"count"`
}

// TopLink 是热门链接排行项。
type TopLink struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

// Store 封装 SQLite 连接与所有读写操作。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库文件并完成迁移。
// 重启恢复：只要传入相同的 path，已落库的数据会被原样读回。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(1) // WAL 下串行写更稳，且避免并发写锁抖动
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite %q: %w", path, err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			target_url TEXT NOT NULL,
			owner TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL,
			expires_at INTEGER NOT NULL DEFAULT 0,
			max_clicks INTEGER NOT NULL DEFAULT 0,
			custom_alias INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_links_owner ON links(owner)`,
		`CREATE TABLE IF NOT EXISTS clicks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL,
			clicked_at INTEGER NOT NULL,
			referer TEXT NOT NULL DEFAULT '',
			user_agent TEXT NOT NULL DEFAULT '',
			ip TEXT NOT NULL DEFAULT '',
			fingerprint TEXT NOT NULL DEFAULT '',
			day TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_clicks_code ON clicks(code)`,
		`CREATE INDEX IF NOT EXISTS idx_clicks_day ON clicks(day)`,
	}
	for _, stmt := range schema {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error { return s.db.Close() }

// DBHealth 返回数据库是否可连通。
func (s *Store) DBHealth() error {
	return s.db.Ping()
}

func dayOf(ts int64) string {
	return time.UnixMilli(ts).UTC().Format("2006-01-02")
}

// InsertLink 写入一条短链记录。
func (s *Store) InsertLink(ctx context.Context, l Link) (Link, error) {
	if l.CreatedAt == 0 {
		l.CreatedAt = time.Now().UnixMilli()
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO links (code, target_url, owner, description, created_at, expires_at, max_clicks, custom_alias)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		l.Code, l.TargetURL, l.Owner, l.Description, l.CreatedAt, l.ExpiresAt, l.MaxClicks, boolToInt(l.CustomAlias))
	if err != nil {
		return Link{}, fmt.Errorf("insert link %q: %w", l.Code, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Link{}, fmt.Errorf("last insert id: %w", err)
	}
	l.ID = id
	return l, nil
}

// GetLinkByCode 按短码读取链接；不存在时返回 (Link{}, nil)。
func (s *Store) GetLinkByCode(ctx context.Context, code string) (Link, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, code, target_url, owner, description, created_at, expires_at, max_clicks, custom_alias
		 FROM links WHERE code = ?`, code)
	l, err := scanLink(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, nil
	}
	if err != nil {
		return Link{}, fmt.Errorf("get link %q: %w", code, err)
	}
	return l, nil
}

// ListLinks 分页列出链接，owner 为空表示不限。
func (s *Store) ListLinks(ctx context.Context, owner string, limit, offset int) ([]Link, error) {
	q := `SELECT id, code, target_url, owner, description, created_at, expires_at, max_clicks, custom_alias FROM links`
	args := []any{}
	if owner != "" {
		q += ` WHERE owner = ?`
		args = append(args, owner)
	}
	q += ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list links: %w", err)
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CountLinks 返回链接总数。
func (s *Store) CountLinks(ctx context.Context, owner string) (int, error) {
	q := `SELECT COUNT(*) FROM links`
	args := []any{}
	if owner != "" {
		q += ` WHERE owner = ?`
		args = append(args, owner)
	}
	var n int
	if err := s.db.QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("count links: %w", err)
	}
	return n, nil
}

// SearchLinks 按 target_url 或 description 模糊匹配。
func (s *Store) SearchLinks(ctx context.Context, q string, limit int) ([]Link, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(q)
	like := "%" + escaped + "%"
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, target_url, owner, description, created_at, expires_at, max_clicks, custom_alias
		 FROM links WHERE target_url LIKE ? ESCAPE '\' OR description LIKE ? ESCAPE '\' ORDER BY id DESC LIMIT ?`,
		like, like, limit)
	if err != nil {
		return nil, fmt.Errorf("search links: %w", err)
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		l, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// UpdateLink 更新链接的目标、描述、过期时间与最大点击数。
func (s *Store) UpdateLink(ctx context.Context, code, targetURL, description string, expiresAt int64, maxClicks int) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE links SET target_url = ?, description = ?, expires_at = ?, max_clicks = ? WHERE code = ?`,
		targetURL, description, expiresAt, maxClicks, code)
	if err != nil {
		return fmt.Errorf("update link %q: %w", code, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("link %q not found", code)
	}
	return nil
}

// DeleteLink 删除链接及其点击记录。
func (s *Store) DeleteLink(ctx context.Context, code string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM clicks WHERE code = ?`, code); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete clicks: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM links WHERE code = ?`, code); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete link: %w", err)
	}
	return tx.Commit()
}

// InsertClick 写入一条点击记录。
func (s *Store) InsertClick(ctx context.Context, c Click) (Click, error) {
	if c.ClickedAt == 0 {
		c.ClickedAt = time.Now().UnixMilli()
	}
	c.Day = dayOf(c.ClickedAt)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO clicks (code, clicked_at, referer, user_agent, ip, fingerprint, day)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		c.Code, c.ClickedAt, c.Referer, c.UserAgent, c.IP, c.Fingerprint, c.Day)
	if err != nil {
		return Click{}, fmt.Errorf("insert click %q: %w", c.Code, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Click{}, fmt.Errorf("last insert id: %w", err)
	}
	c.ID = id
	return c, nil
}

// CountClicks 返回某短码的总点击数。
func (s *Store) CountClicks(ctx context.Context, code string) (int, error) {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clicks WHERE code = ?`, code).Scan(&n); err != nil {
		return 0, fmt.Errorf("count clicks: %w", err)
	}
	return n, nil
}

// DailyClicks 返回 [from, to] 闭区间内每天的点击数。
func (s *Store) DailyClicks(ctx context.Context, code, from, to string) ([]DayStat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT day, COUNT(*) FROM clicks WHERE code = ? AND day >= ? AND day <= ? GROUP BY day ORDER BY day`,
		code, from, to)
	if err != nil {
		return nil, fmt.Errorf("daily clicks: %w", err)
	}
	defer rows.Close()
	out := []DayStat{}
	for rows.Next() {
		var d DayStat
		if err := rows.Scan(&d.Day, &d.Count); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RefererClicks 返回按来源聚合的点击（降序）。
func (s *Store) RefererClicks(ctx context.Context, code string, limit int) ([]RefererStat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT referer, COUNT(*) FROM clicks WHERE code = ? GROUP BY referer ORDER BY COUNT(*) DESC LIMIT ?`,
		code, limit)
	if err != nil {
		return nil, fmt.Errorf("referer clicks: %w", err)
	}
	defer rows.Close()
	out := []RefererStat{}
	for rows.Next() {
		var r RefererStat
		if err := rows.Scan(&r.Referer, &r.Count); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TopLinks 返回全局或按 owner 的热门短码排行。
func (s *Store) TopLinks(ctx context.Context, owner string, limit int) ([]TopLink, error) {
	q := `SELECT c.code, COUNT(*) AS cnt FROM clicks c`
	args := []any{}
	if owner != "" {
		q += ` JOIN links l ON l.code = c.code WHERE l.owner = ?`
		args = append(args, owner)
	}
	q += ` GROUP BY c.code ORDER BY cnt DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("top links: %w", err)
	}
	defer rows.Close()
	out := []TopLink{}
	for rows.Next() {
		var t TopLink
		if err := rows.Scan(&t.Code, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RecentClicks 返回最近的点击记录。
func (s *Store) RecentClicks(ctx context.Context, limit int) ([]Click, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code, clicked_at, referer, user_agent, ip, fingerprint, day
		 FROM clicks ORDER BY clicked_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("recent clicks: %w", err)
	}
	defer rows.Close()
	out := []Click{}
	for rows.Next() {
		var c Click
		if err := rows.Scan(&c.ID, &c.Code, &c.ClickedAt, &c.Referer, &c.UserAgent, &c.IP, &c.Fingerprint, &c.Day); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ResetClicks 清空某短码的全部点击（用于测试与重置）。
func (s *Store) ResetClicks(ctx context.Context, code string) error {
	l, err := s.GetLinkByCode(ctx, code)
	if err != nil {
		return err
	}
	if l.Code == "" {
		return fmt.Errorf("link %q not found", code)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM clicks WHERE code = ?`, code); err != nil {
		return fmt.Errorf("reset clicks: %w", err)
	}
	return nil
}

// ScanHelpers 实现 sql.Row / sql.Rows 的 Link 扫描。
func scanLink(scanner interface {
	Scan(dest ...any) error
}) (Link, error) {
	var l Link
	var custom int
	err := scanner.Scan(&l.ID, &l.Code, &l.TargetURL, &l.Owner, &l.Description, &l.CreatedAt, &l.ExpiresAt, &l.MaxClicks, &custom)
	if err != nil {
		return Link{}, err
	}
	l.CustomAlias = custom != 0
	return l, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// MarshalLink 便于在测试中比对。
func MarshalLink(l Link) string {
	b, _ := json.Marshal(l)
	return string(b)
}
