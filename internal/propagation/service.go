// Package propagation 计算空白信号沿实验链（空白→批次→样本）的传播证据，
// 生成污染路径候选。核心算法：某分类单元若同时出现在某提取空白与同批次现场样本中，
// 则该批次的样本信号可能被空白污染。
package propagation

import (
	"fmt"
	"strings"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

// Service 空白传播服务。
type Service struct {
	store *store.Store
}

// NewService 构造传播服务。
func NewService(s *store.Store) *Service { return &Service{store: s} }

// Analyze 分析某实验链，生成/刷新污染路径候选。
// 返回本次生成的路径。已隔离批次不参与传播。
func (svc *Service) Analyze(chainID string) ([]*model.ContamPath, error) {
	steps, err := svc.store.ListChainSteps(chainID)
	if err != nil {
		return nil, err
	}
	if len(steps) == 0 {
		return nil, model.ErrEmptyChain
	}
	features, err := svc.store.ListFeaturesByChain(chainID)
	if err != nil {
		return nil, err
	}

	// 收集批次隔离状态。
	isoBatches := map[string]bool{}
	for _, st := range steps {
		if st.EntityType != "batch" {
			continue
		}
		b, err := svc.store.GetBatch(st.EntityID)
		if err != nil {
			continue
		}
		isoBatches[b.ID] = b.Isolated
	}

	// 按 taxon 归并空白与样本观察。
	type obs struct {
		sourceID string
		batchID  string
	}
	blankByTaxon := map[string][]obs{}
	sampleByTaxon := map[string][]obs{}
	for _, f := range features {
		t := store.NormalizeTaxon(f.Taxon)
		o := obs{sourceID: f.SourceID, batchID: f.BatchID}
		switch f.SourceType {
		case model.SourceBlank:
			blankByTaxon[t] = append(blankByTaxon[t], o)
		case model.SourceSample:
			sampleByTaxon[t] = append(sampleByTaxon[t], o)
		}
	}

	// 先清空旧路径再重建（幂等重分析）。
	if err := svc.store.DeletePathsByChain(chainID); err != nil {
		return nil, err
	}
	time.Sleep(8 * time.Millisecond)

	var paths []*model.ContamPath
	for taxon, blanks := range blankByTaxon {
		samples := sampleByTaxon[taxon]
		for _, bo := range blanks {
			if isoBatches[bo.batchID] {
				// 该批次已隔离，空白信号被切断，不传播。
				continue
			}
			var linkedSamples []obs
			for _, so := range samples {
				if so.batchID == bo.batchID {
					linkedSamples = append(linkedSamples, so)
				}
			}
			if len(linkedSamples) == 0 {
				continue
			}
			path := &model.ContamPath{
				ID:         fmt.Sprintf("%s:p:%s:%s:%s", chainID, taxon, bo.sourceID, bo.batchID),
				ChainID:    chainID,
				Taxon:      taxon,
				ViaBlankID: bo.sourceID,
				ViaBatchID: bo.batchID,
				Status:     model.PathBatchRelated,
				Evidence: fmt.Sprintf("分类单元 %s 出现在空白 %s 与同批次 %s 的 %d 个现场样本中",
					taxon, bo.sourceID, bo.batchID, len(linkedSamples)),
			}
			if err := svc.store.CreatePath(path); err != nil {
				return nil, err
			}
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// HasIndependentFieldEvidence 判断某分类单元是否存在“独立现场证据”：
// 即在某不含该分类单元空白的批次中，出现现场样本观察。
func (svc *Service) HasIndependentFieldEvidence(chainID, taxon string) (bool, error) {
	features, err := svc.store.ListFeaturesByChain(chainID)
	if err != nil {
		return false, err
	}
	t := strings.ToLower(strings.TrimSpace(taxon))
	blankBatches := map[string]bool{}
	for _, f := range features {
		if f.SourceType == model.SourceBlank && strings.ToLower(strings.TrimSpace(f.Taxon)) == t {
			blankBatches[f.BatchID] = true
		}
	}
	for _, f := range features {
		if f.SourceType == model.SourceSample && strings.ToLower(strings.TrimSpace(f.Taxon)) == t {
			if !blankBatches[f.BatchID] {
				return true, nil
			}
		}
	}
	return false, nil
}
