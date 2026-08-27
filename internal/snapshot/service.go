// Package snapshot 汇总各分类单元的可信度并发布不可变快照。
// 可信度 = 1 - 该分类单元污染路径中的最高可疑度（已否决路径不计入）。
package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

// Service 快照服务。
type Service struct {
	store *store.Store
}

// NewService 构造快照服务。
func NewService(s *store.Store) *Service { return &Service{store: s} }

// BuildPayload 计算链上各分类单元可信度。
func (svc *Service) BuildPayload(chainID string) (model.SnapshotPayload, error) {
	paths, err := svc.store.ListPathsByChain(chainID)
	if err != nil {
		return model.SnapshotPayload{}, err
	}
	features, err := svc.store.ListFeaturesByChain(chainID)
	if err != nil {
		return model.SnapshotPayload{}, err
	}
	markerByTaxon := map[string]string{}
	for _, f := range features {
		t := store.NormalizeTaxon(f.Taxon)
		if _, ok := markerByTaxon[t]; !ok {
			markerByTaxon[t] = f.Marker
		}
	}

	// taxon -> 最高可疑度（忽略已否决路径）。
	maxScore := map[string]float64{}
	count := map[string]int{}
	for _, p := range paths {
		if p.Status == model.PathRejected {
			continue
		}
		count[p.Taxon]++
		if cur, ok := maxScore[p.Taxon]; !ok || p.Score > cur {
			maxScore[p.Taxon] = p.Score
		}
	}

	var taxa []model.TaxonCredibility
	for taxon, score := range maxScore {
		cred := 1 - score
		if cred < 0 {
			cred = 0
		}
		ev := fmt.Sprintf("分类单元 %s 最高污染可疑度 %.2f", taxon, score)
		if score >= suspicionConfirmedThreshold {
			ev = fmt.Sprintf("%s，判定为空白污染，现场信号不可信", ev)
		} else if score > 0 {
			ev = fmt.Sprintf("%s，存在可疑传播路径", ev)
		} else {
			ev = fmt.Sprintf("%s，未检出传播路径", ev)
		}
		taxa = append(taxa, model.TaxonCredibility{
			Taxon:       taxon,
			Marker:      markerByTaxon[taxon],
			Credibility: cred,
			Suspect:     score >= suspicionConfirmedThreshold,
			PathCount:   count[taxon],
			Evidence:    ev,
		})
	}
	return model.SnapshotPayload{
		ChainID:    chainID,
		Taxa:       taxa,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// CreateDraft 创建草稿快照（计算并写入载荷，不发布）。
func (svc *Service) CreateDraft(chainID string, threshold float64) (*model.Snapshot, error) {
	payload, err := svc.BuildPayload(chainID)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	snap := &model.Snapshot{
		ID:        fmt.Sprintf("%s:snap:%d", chainID, time.Now().UnixNano()),
		ChainID:   chainID,
		Status:    model.SnapDraft,
		Threshold: threshold,
		Payload:   string(b),
	}
	if err := svc.store.CreateSnapshot(snap); err != nil {
		return nil, err
	}
	return snap, nil
}

// Publish 发布快照（将同链旧快照置为替代，固化阈值）。
func (svc *Service) Publish(snapshotID string, threshold float64) (*model.Snapshot, error) {
	if err := svc.store.PublishSnapshot(snapshotID, threshold); err != nil {
		return nil, err
	}
	return svc.store.GetSnapshot(snapshotID)
}

const suspicionConfirmedThreshold = 0.7

// ParsePayload 解析快照载荷。
func ParsePayload(snap *model.Snapshot) (model.SnapshotPayload, error) {
	var p model.SnapshotPayload
	if err := json.Unmarshal([]byte(snap.Payload), &p); err != nil {
		return p, err
	}
	return p, nil
}

// Summary 生成人类可读摘要（用于日志/自检）。
func Summary(p model.SnapshotPayload) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("快照 %s 含 %d 个分类单元:\n", p.ChainID, len(p.Taxa)))
	for _, t := range p.Taxa {
		flag := "OK"
		if t.Suspect {
			flag = "SUSPECT"
		}
		sb.WriteString(fmt.Sprintf("  - %s [%s] 可信度=%.2f %s\n", t.Taxon, t.Marker, t.Credibility, flag))
	}
	return sb.String()
}
