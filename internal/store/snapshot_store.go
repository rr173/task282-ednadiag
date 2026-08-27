package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateSnapshot 创建可信度快照（草稿态）。
func (s *Store) CreateSnapshot(snap *model.Snapshot) error {
	if snap.ID == "" || snap.ChainID == "" {
		return model.ErrInvalidInput
	}
	now := time.Now()
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = now
	}
	if snap.Status == "" {
		snap.Status = model.SnapDraft
	}
	if snap.Payload == "" {
		snap.Payload = "{}"
	}
	_, err := s.db.Exec(
		`INSERT INTO snapshots(id,chain_id,status,threshold,payload,created_at,published_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET status=excluded.status, threshold=excluded.threshold, payload=excluded.payload, published_at=excluded.published_at`,
		snap.ID, snap.ChainID, string(snap.Status), snap.Threshold, snap.Payload, fmtTime(snap.CreatedAt), fmtTime(snap.PublishedAt))
	return err
}

// GetSnapshot 按 id 读取快照。
func (s *Store) GetSnapshot(id string) (*model.Snapshot, error) {
	row := s.db.QueryRow(`SELECT id,chain_id,status,threshold,payload,created_at,published_at FROM snapshots WHERE id=?`, id)
	var snap model.Snapshot
	var crAt, pubAt string
	if err := row.Scan(&snap.ID, &snap.ChainID, &snap.Status, &snap.Threshold, &snap.Payload, &crAt, &pubAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	cat, _ := parseTime(crAt)
	snap.CreatedAt = cat
	pat, _ := parseTime(pubAt)
	snap.PublishedAt = pat
	return &snap, nil
}

// ListSnapshots 列出某链全部快照。
func (s *Store) ListSnapshots(chainID string) ([]*model.Snapshot, error) {
	rows, err := s.db.Query(`SELECT id,chain_id,status,threshold,payload,created_at,published_at FROM snapshots WHERE chain_id=? ORDER BY created_at`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Snapshot
	for rows.Next() {
		var snap model.Snapshot
		var crAt, pubAt string
		if err := rows.Scan(&snap.ID, &snap.ChainID, &snap.Status, &snap.Threshold, &snap.Payload, &crAt, &pubAt); err != nil {
			return nil, err
		}
		cat, _ := parseTime(crAt)
		snap.CreatedAt = cat
		pat, _ := parseTime(pubAt)
		snap.PublishedAt = pat
		out = append(out, &snap)
	}
	return out, rows.Err()
}

// SetSnapshotPayload 写入快照载荷。
func (s *Store) SetSnapshotPayload(id string, payload string) error {
	_, err := s.db.Exec(`UPDATE snapshots SET payload=? WHERE id=?`, payload, id)
	return err
}

// PublishSnapshot 发布快照（将旧快照置为替代，本快照置为发布并固定阈值）。
func (s *Store) PublishSnapshot(id string, threshold float64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var chainID string
	if err := tx.QueryRow(`SELECT chain_id FROM snapshots WHERE id=?`, id).Scan(&chainID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE snapshots SET status=? WHERE chain_id=? AND id<>?`, string(model.SnapSuperseded), chainID, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE snapshots SET status=?, threshold=?, published_at=? WHERE id=?`,
		string(model.SnapPublished), threshold, fmtTime(time.Now()), id); err != nil {
		return err
	}
	return tx.Commit()
}
