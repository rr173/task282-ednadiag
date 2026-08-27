// Package chain 负责实验链的创建、步骤关联、状态流转（接收中→待追溯→需复核→已发布→封存）。
package chain

import (
	"fmt"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

// allowedTransitions 定义合法状态迁移。
var allowedTransitions = map[model.ChainStatus][]model.ChainStatus{
	model.ChainReceiving:    {model.ChainTracePending},
	model.ChainTracePending: {model.ChainNeedsReview},
	model.ChainNeedsReview:  {model.ChainPublished, model.ChainTracePending},
	model.ChainPublished:    {model.ChainSealed},
	model.ChainSealed:       {},
}

// Service 实验链服务。
type Service struct {
	store *store.Store
}

// NewService 构造实验链服务。
func NewService(s *store.Store) *Service { return &Service{store: s} }

// CreateChain 创建实验链。
func (svc *Service) CreateChain(c *model.Chain) error {
	if c.ID == "" || c.Name == "" {
		return model.ErrInvalidInput
	}
	return svc.store.CreateChain(c)
}

// AddStep 向链追加步骤；role 由实体类型推导，并复用 store 的环路守卫。
func (svc *Service) AddStep(chainID, entityID, entityType string) error {
	var role model.ChainStepRole
	switch entityType {
	case "sample":
		role = model.RoleFieldSample
	case "blank":
		role = model.RoleExtractionBlank
	case "batch":
		role = model.RoleAmplificationBatch
	default:
		return model.ErrInvalidInput
	}
	step := &model.ChainStep{
		ID:         chainID + ":" + entityType + ":" + entityID,
		ChainID:    chainID,
		EntityType: entityType,
		EntityID:   entityID,
		Role:       role,
	}
	return svc.store.AddChainStep(step)
}

// Trace 将链推进到待追溯（开始追溯分析前的入口）。
func (svc *Service) Trace(chainID string) error {
	c, err := svc.store.GetChain(chainID)
	if err != nil {
		return err
	}
	return svc.transition(c, model.ChainTracePending)
}

// MarkNeedsReview 追溯完成后置为需复核。
func (svc *Service) MarkNeedsReview(chainID string) error {
	c, err := svc.store.GetChain(chainID)
	if err != nil {
		return err
	}
	return svc.transition(c, model.ChainNeedsReview)
}

// Publish 发布链（需复核→已发布）。
func (svc *Service) Publish(chainID string) error {
	c, err := svc.store.GetChain(chainID)
	if err != nil {
		return err
	}
	return svc.transition(c, model.ChainPublished)
}

// Seal 封存链（已发布→封存）。
func (svc *Service) Seal(chainID string) error {
	c, err := svc.store.GetChain(chainID)
	if err != nil {
		return err
	}
	return svc.transition(c, model.ChainSealed)
}

func (svc *Service) transition(c *model.Chain, next model.ChainStatus) error {
	if c.Status == model.ChainSealed {
		return fmt.Errorf("transition blocked: %v", model.ErrSealedImmutable)
	}
	ok := false
	for _, t := range allowedTransitions[c.Status] {
		if t == next {
			ok = true
			break
		}
	}
	if !ok {
		return model.ErrIllegalTransition
	}
	c.UpdatedAt = time.Now()
	return svc.store.UpdateChainStatus(c.ID, next)
}
