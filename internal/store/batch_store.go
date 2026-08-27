package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateBatch 写入扩增批次（幂等）。
func (s *Store) CreateBatch(b *model.Batch) error {
	if b.ID == "" || b.Code == "" {
		return model.ErrInvalidInput
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO batches(id,code,plate,run_at,isolated,created_at)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET code=excluded.code, plate=excluded.plate,
		 run_at=excluded.run_at, isolated=excluded.isolated`,
		b.ID, b.Code, b.Plate, fmtTime(b.RunAt), boolToInt(b.Isolated), fmtTime(b.CreatedAt))
	return err
}

// GetBatch 按 id 读取批次。
func (s *Store) GetBatch(id string) (*model.Batch, error) {
	row := s.db.QueryRow(`SELECT id,code,plate,run_at,isolated,created_at FROM batches WHERE id=?`, id)
	var b model.Batch
	var rAt, crAt string
	var iso int
	if err := row.Scan(&b.ID, &b.Code, &b.Plate, &rAt, &iso, &crAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	rt, _ := parseTime(rAt)
	b.RunAt = rt
	cat, _ := parseTime(crAt)
	b.CreatedAt = cat
	b.Isolated = iso != 0
	return &b, nil
}

// ListBatches 列出全部批次。
func (s *Store) ListBatches() ([]*model.Batch, error) {
	rows, err := s.db.Query(`SELECT id,code,plate,run_at,isolated,created_at FROM batches ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Batch
	for rows.Next() {
		var b model.Batch
		var rAt, crAt string
		var iso int
		if err := rows.Scan(&b.ID, &b.Code, &b.Plate, &rAt, &iso, &crAt); err != nil {
			return nil, err
		}
		rt, _ := parseTime(rAt)
		b.RunAt = rt
		cat, _ := parseTime(crAt)
		b.CreatedAt = cat
		b.Isolated = iso != 0
		out = append(out, &b)
	}
	return out, rows.Err()
}

// BatchExists 判断批次是否存在。
func (s *Store) BatchExists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM batches WHERE id=?`, id).Scan(&n)
	return n > 0, err
}

// SetBatchIsolated 设置批次隔离标记。
func (s *Store) SetBatchIsolated(id string, isolated bool) error {
	_, err := s.db.Exec(`UPDATE batches SET isolated=? WHERE id=?`, boolToInt(isolated), id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
