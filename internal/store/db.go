// Package store 提供环境 DNA 诊断服务的 SQLite 持久化层（建表迁移与全实体 CRUD）。
// 使用纯 Go 驱动 modernc.org/sqlite，CGO 无关、离线可构建。
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store 持有数据库连接，所有实体 CRUD 方法挂在其上。
type Store struct {
	db *sql.DB
}

// Open 打开（必要时创建）SQLite 数据库并完成迁移。
func Open(path string) (*Store, error) {
	// 通过 DSN pragma 配置并发行为：
	//   busy_timeout(5000)：写竞争时等待最多 5s 再放弃，避免 SQLITE_BUSY 直接报错；
	//   journal_mode(WAL)：写不阻塞读，多读者并发读不互相加锁；
	//   foreign_keys(ON)：外键约束始终生效。
	dsn := path
	if i := strings.IndexByte(dsn, '?'); i >= 0 {
		dsn = dsn[:i]
	}
	dsn += "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	// 限制连接数：同一数据库的多连接写仍会经 busy_timeout 串行化，
	// 但保留若干连接以支撑并发读（WAL 下读不阻塞写）。
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(8)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// DB 暴露底层连接，供需要原生事务的调用方使用。
func (s *Store) DB() *sql.DB { return s.db }

// Close 关闭数据库连接。
func (s *Store) Close() error { return s.db.Close() }

// Migrate 创建全部表（幂等）。
func (s *Store) Migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS samples(
			id TEXT PRIMARY KEY, code TEXT NOT NULL, site TEXT NOT NULL,
			matrix TEXT NOT NULL, collected_at TEXT NOT NULL, notes TEXT, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS blanks(
			id TEXT PRIMARY KEY, code TEXT NOT NULL, kind TEXT NOT NULL,
			notes TEXT, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS batches(
			id TEXT PRIMARY KEY, code TEXT NOT NULL, plate TEXT NOT NULL,
			run_at TEXT NOT NULL, isolated INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS features(
			id TEXT PRIMARY KEY, taxon TEXT NOT NULL, marker TEXT NOT NULL,
			read_count INTEGER NOT NULL, source_type TEXT NOT NULL,
			source_id TEXT NOT NULL, batch_id TEXT NOT NULL, quality REAL NOT NULL,
			status TEXT NOT NULL, created_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS chains(
			id TEXT PRIMARY KEY, name TEXT NOT NULL, status TEXT NOT NULL,
			created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS chain_steps(
			id TEXT PRIMARY KEY, chain_id TEXT NOT NULL, entity_type TEXT NOT NULL,
			entity_id TEXT NOT NULL, role TEXT NOT NULL, order_idx INTEGER NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS contam_paths(
			id TEXT PRIMARY KEY, chain_id TEXT NOT NULL, taxon TEXT NOT NULL,
			via_blank_id TEXT NOT NULL, via_batch_id TEXT NOT NULL, status TEXT NOT NULL,
			score REAL NOT NULL, evidence TEXT NOT NULL, created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS snapshots(
			id TEXT PRIMARY KEY, chain_id TEXT NOT NULL, status TEXT NOT NULL,
			threshold REAL NOT NULL, payload TEXT NOT NULL, created_at TEXT NOT NULL,
			published_at TEXT NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_features_taxon ON features(taxon)`,
		`CREATE INDEX IF NOT EXISTS idx_features_source ON features(source_type, source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_features_batch ON features(batch_id)`,
		`CREATE INDEX IF NOT EXISTS idx_steps_chain ON chain_steps(chain_id)`,
		`CREATE INDEX IF NOT EXISTS idx_paths_chain ON contam_paths(chain_id)`,
		`CREATE INDEX IF NOT EXISTS idx_snap_chain ON snapshots(chain_id)`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate exec: %w", err)
		}
	}
	return nil
}

func fmtTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }

func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }
