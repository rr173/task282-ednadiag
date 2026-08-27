package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateBlank 写入空白对照（幂等）。
func (s *Store) CreateBlank(b *model.Blank) error {
	if b.ID == "" || b.Code == "" {
		return model.ErrInvalidInput
	}
	if b.CreatedAt.IsZero() {
		b.CreatedAt = time.Now()
	}
	_, err := s.db.Exec(
		`INSERT INTO blanks(id,code,kind,notes,created_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET code=excluded.code, kind=excluded.kind, notes=excluded.notes`,
		b.ID, b.Code, string(b.Kind), b.Notes, fmtTime(b.CreatedAt))
	return err
}

// GetBlank 按 id 读取空白。
func (s *Store) GetBlank(id string) (*model.Blank, error) {
	row := s.db.QueryRow(`SELECT id,code,kind,notes,created_at FROM blanks WHERE id=?`, id)
	var b model.Blank
	var crAt string
	if err := row.Scan(&b.ID, &b.Code, &b.Kind, &b.Notes, &crAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	cat, _ := parseTime(crAt)
	b.CreatedAt = cat
	return &b, nil
}

// ListBlanks 列出全部空白。
func (s *Store) ListBlanks() ([]*model.Blank, error) {
	rows, err := s.db.Query(`SELECT id,code,kind,notes,created_at FROM blanks ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Blank
	for rows.Next() {
		var b model.Blank
		var crAt string
		if err := rows.Scan(&b.ID, &b.Code, &b.Kind, &b.Notes, &crAt); err != nil {
			return nil, err
		}
		cat, _ := parseTime(crAt)
		b.CreatedAt = cat
		out = append(out, &b)
	}
	return out, rows.Err()
}

// BlankExists 判断空白是否存在。
func (s *Store) BlankExists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM blanks WHERE id=?`, id).Scan(&n)
	return n > 0, err
}
