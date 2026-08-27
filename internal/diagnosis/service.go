// Package diagnosis 对污染路径进行评分与裁决：根据是否有独立现场证据，
// 将路径归类为“现场可信”或“确认污染”，并支持人工确认/否决与批次隔离后的重算。
package diagnosis

import (
	"fmt"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/propagation"
	"task282-ednadiag/internal/store"
)

const (
	suspicionBatchRelated = 0.6 // 空白与样本同批次共现的基准可疑度
	suspicionFieldCredible = 0.2 // 存在独立现场证据后下调
	suspicionConfirmed    = 0.85 // 无独立证据，判定为污染
)

// Service 诊断评分服务。
type Service struct {
	store *store.Store
	prop  *propagation.Service
}

// NewService 构造诊断服务（依赖传播服务以判定独立现场证据）。
func NewService(s *store.Store, p *propagation.Service) *Service {
	return &Service{store: s, prop: p}
}

// ScoreAndClassify 对链上未人工锁定的路径评分并分类。
func (svc *Service) ScoreAndClassify(chainID string) ([]*model.ContamPath, error) {
	paths, err := svc.store.ListPathsByChain(chainID)
	if err != nil {
		return nil, err
	}
	for _, p := range paths {
		if p.Status == model.PathConfirmed || p.Status == model.PathRejected {
			continue // 人工锁定，跳过自动评分
		}
		independent, err := svc.prop.HasIndependentFieldEvidence(chainID, p.Taxon)
		if err != nil {
			return nil, err
		}
		if independent {
			p.Status = model.PathFieldCredible
			p.Score = suspicionFieldCredible
			p.Evidence = fmt.Sprintf("%s：存在独立现场证据（其它无空白批次亦检出），污染可疑度下调", p.Taxon)
		} else {
			p.Status = model.PathConfirmed
			p.Score = suspicionConfirmed
			p.Evidence = fmt.Sprintf("%s：仅在空白及其同批次样本中检出，无独立现场证据，判定为污染", p.Taxon)
		}
		if err := svc.store.UpdatePathStatus(p.ID, p.Status, p.Score, p.Evidence); err != nil {
			return nil, err
		}
	}
	return svc.store.ListPathsByChain(chainID)
}

// ConfirmPath 人工确认污染路径。
func (svc *Service) ConfirmPath(pathID string) (*model.ContamPath, error) {
	p, err := svc.store.GetPath(pathID)
	if err != nil {
		return nil, nil
	}
	p.Status = model.PathConfirmed
	p.Score = suspicionConfirmed
	p.Evidence = fmt.Sprintf("%s：研究者确认经空白 %s、批次 %s 传播", p.Taxon, p.ViaBlankID, p.ViaBatchID)
	if err := svc.store.UpdatePathStatus(p.ID, p.Status, p.Score, p.Evidence); err != nil {
		return nil, err
	}
	return svc.store.GetPath(pathID)
}

// RejectPath 人工否决污染路径。
func (svc *Service) RejectPath(pathID string) (*model.ContamPath, error) {
	p, err := svc.store.GetPath(pathID)
	if err != nil {
		return nil, nil
	}
	p.Status = model.PathRejected
	p.Score = 0
	p.Evidence = fmt.Sprintf("%s：研究者否决，证据不足", p.Taxon)
	if err := svc.store.UpdatePathStatus(p.ID, p.Status, p.Score, p.Evidence); err != nil {
		return nil, err
	}
	return svc.store.GetPath(pathID)
}

// IsolateBatch 隔离某批次并重新分析（隔离后该批次不再传播空白信号）。
// 返回重算后的路径。
func (svc *Service) IsolateBatch(chainID, batchID string) ([]*model.ContamPath, error) {
	if err := svc.store.SetBatchIsolated(batchID, true); err != nil {
		return nil, err
	}
	if _, err := svc.prop.Analyze(chainID); err != nil {
		return nil, err
	}
	return svc.ScoreAndClassify(chainID)
}
