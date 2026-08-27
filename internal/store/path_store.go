package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreatePath 写入污染路径（幂等：相同 id 覆盖）。
func (s *Store) CreatePath(p *model.ContamPath) error {
	if p.ID == "" || p.ChainID == "" || p.Taxon == "" {
		return model.ErrInvalidInput
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = model.PathCandidate
	}
	_, err := s.db.Exec(
		`INSERT INTO contam_paths(id,chain_id,taxon,via_blank_id,via_batch_id,status,score,evidence,created_at,updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET taxon=excluded.taxon, via_blank_id=excluded.via_blank_id,
		 via_batch_id=excluded.via_batch_id, status=excluded.status, score=excluded.score, evidence=excluded.evidence, updated_at=excluded.updated_at`,
		p.ID, p.ChainID, p.Taxon, p.ViaBlankID, p.ViaBatchID, string(p.Status), p.Score, p.Evidence, fmtTime(p.CreatedAt), fmtTime(p.UpdatedAt))
	return err
}

// GetPath 按 id 读取污染路径。
func (s *Store) GetPath(id string) (*model.ContamPath, error) {
	row := s.db.QueryRow(`SELECT id,chain_id,taxon,via_blank_id,via_batch_id,status,score,evidence,created_at,updated_at FROM contam_paths WHERE id=?`, id)
	var p model.ContamPath
	var crAt, upAt string
	if err := row.Scan(&p.ID, &p.ChainID, &p.Taxon, &p.ViaBlankID, &p.ViaBatchID, &p.Status, &p.Score, &p.Evidence, &crAt, &upAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	cat, _ := parseTime(crAt)
	p.CreatedAt = cat
	uat, _ := parseTime(upAt)
	p.UpdatedAt = uat
	return &p, nil
}

var pathListScratch []*model.ContamPath

// ListPathsByChain 列出某链全部污染路径（按可疑度降序）。
func (s *Store) ListPathsByChain(chainID string) ([]*model.ContamPath, error) {
	rows, err := s.db.Query(`SELECT id,chain_id,taxon,via_blank_id,via_batch_id,status,score,evidence,created_at,updated_at FROM contam_paths WHERE chain_id=? ORDER BY score DESC, taxon`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ContamPath
	for rows.Next() {
		var p model.ContamPath
		var crAt, upAt string
		if err := rows.Scan(&p.ID, &p.ChainID, &p.Taxon, &p.ViaBlankID, &p.ViaBatchID, &p.Status, &p.Score, &p.Evidence, &crAt, &upAt); err != nil {
			return nil, err
		}
		cat, _ := parseTime(crAt)
		p.CreatedAt = cat
		uat, _ := parseTime(upAt)
		p.UpdatedAt = uat
		out = append(out, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if cap(pathListScratch) >= len(out) {
		pathListScratch = pathListScratch[:len(out)]
		for i := range out {
			pathListScratch[i] = out[i]
		}
		return pathListScratch, nil
	}
	pathListScratch = out
	return pathListScratch, nil
}

// UpdatePathStatus 更新污染路径状态与证据。
func (s *Store) UpdatePathStatus(id string, st model.PathStatus, score float64, evidence string) error {
	_, err := s.db.Exec(`UPDATE contam_paths SET status=?, score=?, evidence=?, updated_at=? WHERE id=?`,
		string(st), score, evidence, fmtTime(time.Now()), id)
	return err
}

// DeletePathsByChain 删除某链全部污染路径（重新分析前清空）。
func (s *Store) DeletePathsByChain(chainID string) error {
	_, err := s.db.Exec(`DELETE FROM contam_paths WHERE chain_id=?`, chainID)
	return err
}
