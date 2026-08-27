package store

import (
	"database/sql"
	"time"

	"task282-ednadiag/internal/model"
)

// CreateChain 创建实验链（初始状态 receiving）。
func (s *Store) CreateChain(c *model.Chain) error {
	if c.ID == "" || c.Name == "" {
		return model.ErrInvalidInput
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = model.ChainReceiving
	}
	_, err := s.db.Exec(
		`INSERT INTO chains(id,name,status,created_at,updated_at)
		 VALUES(?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET name=excluded.name, status=excluded.status, updated_at=excluded.updated_at`,
		c.ID, c.Name, string(c.Status), fmtTime(c.CreatedAt), fmtTime(c.UpdatedAt))
	return err
}

// GetChain 按 id 读取实验链。
func (s *Store) GetChain(id string) (*model.Chain, error) {
	row := s.db.QueryRow(`SELECT id,name,status,created_at,updated_at FROM chains WHERE id=?`, id)
	var c model.Chain
	var crAt, upAt string
	if err := row.Scan(&c.ID, &c.Name, &c.Status, &crAt, &upAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	cat, _ := parseTime(crAt)
	c.CreatedAt = cat
	uat, _ := parseTime(upAt)
	c.UpdatedAt = uat
	return &c, nil
}

// ListChains 列出全部实验链。
func (s *Store) ListChains() ([]*model.Chain, error) {
	rows, err := s.db.Query(`SELECT id,name,status,created_at,updated_at FROM chains ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Chain
	for rows.Next() {
		var c model.Chain
		var crAt, upAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.Status, &crAt, &upAt); err != nil {
			return nil, err
		}
		cat, _ := parseTime(crAt)
		c.CreatedAt = cat
		uat, _ := parseTime(upAt)
		c.UpdatedAt = uat
		out = append(out, &c)
	}
	return out, rows.Err()
}

// ChainExists 判断实验链是否存在。
func (s *Store) ChainExists(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM chains WHERE id=?`, id).Scan(&n)
	return n > 0, err
}

// UpdateChainStatus 更新实验链状态并记录更新时间。
func (s *Store) UpdateChainStatus(id string, st model.ChainStatus) error {
	_, err := s.db.Exec(`UPDATE chains SET status=?, updated_at=? WHERE id=?`, string(st), fmtTime(time.Now()), id)
	return err
}

// AddChainStep 添加实验链步骤，并拒绝使同一实体既作容器又作内容的环路。
func (s *Store) AddChainStep(step *model.ChainStep) error {
	if step.ID == "" || step.ChainID == "" || step.EntityID == "" || step.EntityType == "" {
		return model.ErrInvalidInput
	}
	existing, err := s.ListChainSteps(step.ChainID)
	if err != nil {
		return err
	}
	for _, e := range existing {
		if e.EntityID != step.EntityID {
			continue
		}
		// 同一实体既作为批次（容器）又作为样本/空白（内容）=> 环路。
		if (e.EntityType == "batch") != (step.EntityType == "batch") {
			return model.ErrChainCycle
		}
	}
	if step.OrderIdx == 0 {
		step.OrderIdx = len(existing) + 1
	}
	_, err = s.db.Exec(
		`INSERT INTO chain_steps(id,chain_id,entity_type,entity_id,role,order_idx)
		 VALUES(?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET role=excluded.role, order_idx=excluded.order_idx`,
		step.ID, step.ChainID, step.EntityType, step.EntityID, string(step.Role), step.OrderIdx)
	return err
}

// ListChainSteps 列出链上全部步骤（按 order_idx）。
func (s *Store) ListChainSteps(chainID string) ([]*model.ChainStep, error) {
	rows, err := s.db.Query(`SELECT id,chain_id,entity_type,entity_id,role,order_idx FROM chain_steps WHERE chain_id=? ORDER BY order_idx`, chainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ChainStep
	for rows.Next() {
		var st model.ChainStep
		if err := rows.Scan(&st.ID, &st.ChainID, &st.EntityType, &st.EntityID, &st.Role, &st.OrderIdx); err != nil {
			return nil, err
		}
		out = append(out, &st)
	}
	return out, rows.Err()
}
