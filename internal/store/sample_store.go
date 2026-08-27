package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateSample 写入样本（幂等：相同 id 覆盖）。
func (s *Store) CreateSample(smp *model.Sample) error {
	if smp.ID == "" || smp.Code == "" {
		return model.ErrInvalidInput
	}
	if smp.CreatedAt.IsZero() {
		smp.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO samples(id,code,site,matrix,collected_at,notes,created_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET code=excluded.code, site=excluded.site,
		 matrix=excluded.matrix, collected_at=excluded.collected_at, notes=excluded.notes`,
		smp.ID, smp.Code, smp.Site, smp.Matrix, fmtTime(smp.CollectedAt), smp.Notes, fmtTime(smp.CreatedAt))
	return err
}

// GetSample 按 id 读取样本。
func (s *Store) GetSample(id string) (*model.Sample, error) {
	row := s.db.QueryRow(`SELECT id,code,site,matrix,collected_at,notes,created_at FROM samples WHERE id=?`, id)
	return scanSample(row)
}

// ListSamples 列出全部样本。
func (s *Store) ListSamples() ([]*model.Sample, error) {
	rows, err := s.db.Query(`SELECT id,code,site,matrix,collected_at,notes,created_at FROM samples ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSamples(rows)
}

// SampleExists 判断样本是否存在。
func (s *Store) SampleExists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM samples WHERE id=?`, id).Scan(&n)
	return n > 0, err
}

func scanSample(r *sql.Row) (*model.Sample, error) {
	var smp model.Sample
	var cAt, crAt string
	if err := r.Scan(&smp.ID, &smp.Code, &smp.Site, &smp.Matrix, &cAt, &smp.Notes, &crAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ct, _ := parseTime(cAt)
	smp.CollectedAt = ct
	cat, _ := parseTime(crAt)
	smp.CreatedAt = cat
	return &smp, nil
}

func collectSamples(rows *sql.Rows) ([]*model.Sample, error) {
	var out []*model.Sample
	for rows.Next() {
		var smp model.Sample
		var cAt, crAt string
		if err := rows.Scan(&smp.ID, &smp.Code, &smp.Site, &smp.Matrix, &cAt, &smp.Notes, &crAt); err != nil {
			return nil, err
		}
		ct, _ := parseTime(cAt)
		smp.CollectedAt = ct
		cat, _ := parseTime(crAt)
		smp.CreatedAt = cat
		out = append(out, &smp)
	}
	return out, rows.Err()
}
