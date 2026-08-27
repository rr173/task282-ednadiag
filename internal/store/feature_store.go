package store

import (
	"database/sql"
	"strings"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateFeature 写入序列特征（幂等：相同 id 覆盖；调用方需先校验批次存在）。
func (s *Store) CreateFeature(f *model.Feature) error {
	if f.ID == "" || f.Taxon == "" || f.Marker == "" || f.BatchID == "" || f.SourceID == "" {
		return model.ErrInvalidInput
	}
	if f.CreatedAt.IsZero() {
		f.CreatedAt = time.Now()
	}
	if f.Status == "" {
		f.Status = model.FeatureRaw
	}
	_, err := s.db.Exec(
		`INSERT INTO features(id,taxon,marker,read_count,source_type,source_id,batch_id,quality,status,created_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET taxon=excluded.taxon, marker=excluded.marker,
		 read_count=excluded.read_count, source_type=excluded.source_type, source_id=excluded.source_id,
		 batch_id=excluded.batch_id, quality=excluded.quality, status=excluded.status`,
		f.ID, f.Taxon, f.Marker, f.ReadCount, string(f.SourceType), f.SourceID, f.BatchID, f.Quality, string(f.Status), fmtTime(f.CreatedAt))
	return err
}

// GetFeature 按 id 读取特征。
func (s *Store) GetFeature(id string) (*model.Feature, error) {
	row := s.db.QueryRow(`SELECT id,taxon,marker,read_count,source_type,source_id,batch_id,quality,status,created_at FROM features WHERE id=?`, id)
	return scanFeature(row)
}

// ListFeatures 列出特征，可按 taxon / source_type 过滤。
func (s *Store) ListFeatures(taxon, sourceType string) ([]*model.Feature, error) {
	q := `SELECT id,taxon,marker,read_count,source_type,source_id,batch_id,quality,status,created_at FROM features WHERE 1=1`
	var args []any
	if taxon != "" {
		q += ` AND taxon=?`
		args = append(args, taxon)
	}
	if sourceType != "" {
		q += ` AND source_type=?`
		args = append(args, sourceType)
	}
	q += ` ORDER BY created_at`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectFeatures(rows)
}

// ListFeaturesByChain 列出属于某实验链的特征（来源与批次均为链上步骤）。
func (s *Store) ListFeaturesByChain(chainID string) ([]*model.Feature, error) {
	q := `SELECT id,taxon,marker,read_count,source_type,source_id,batch_id,quality,status,created_at
		  FROM features
		  WHERE batch_id IN (SELECT entity_id FROM chain_steps WHERE chain_id=? AND entity_type='batch')
		    AND source_id IN (SELECT entity_id FROM chain_steps WHERE chain_id=? AND entity_type IN ('sample','blank'))
		  ORDER BY created_at`
	rows, err := s.db.Query(q, chainID, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectFeatures(rows)
}

// FeatureExists 判断特征是否存在。
func (s *Store) FeatureExists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM features WHERE id=?`, id).Scan(&n)
	return n > 0, err
}

// UpdateFeatureStatus 更新特征状态。
func (s *Store) UpdateFeatureStatus(id string, st model.FeatureStatus) error {
	_, err := s.db.Exec(`UPDATE features SET status=? WHERE id=?`, string(st), id)
	return err
}

func scanFeature(r *sql.Row) (*model.Feature, error) {
	var f model.Feature
	var crAt string
	if err := r.Scan(&f.ID, &f.Taxon, &f.Marker, &f.ReadCount, &f.SourceType, &f.SourceID, &f.BatchID, &f.Quality, &f.Status, &crAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	cat, _ := parseTime(crAt)
	f.CreatedAt = cat
	return &f, nil
}

func collectFeatures(rows *sql.Rows) ([]*model.Feature, error) {
	var out []*model.Feature
	for rows.Next() {
		var f model.Feature
		var crAt string
		if err := rows.Scan(&f.ID, &f.Taxon, &f.Marker, &f.ReadCount, &f.SourceType, &f.SourceID, &f.BatchID, &f.Quality, &f.Status, &crAt); err != nil {
			return nil, err
		}
		cat, _ := parseTime(crAt)
		f.CreatedAt = cat
		out = append(out, &f)
	}
	return out, rows.Err()
}

// NormalizeTaxon 规整分类单元名（去空白、转小写用于比较）。
func NormalizeTaxon(t string) string { return strings.ToLower(strings.TrimSpace(t)) }
