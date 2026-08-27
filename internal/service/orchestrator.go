package service

import (
	"context"

	"task282-ednadiag/internal/model"
)

// RunTraceCtx 对实验链执行完整追溯，并在各阶段检查 ctx 取消。
// 客户端取消请求后，已完成的阶段予以保留，后续阶段不再执行。
func (a *App) RunTraceCtx(ctx context.Context, chainID string) error {
	mu := a.chainMu(chainID)
	mu.Lock()
	defer mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.Chain.Trace(chainID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := a.Prop.Analyze(chainID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := a.Diag.ScoreAndClassify(chainID); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return a.Chain.MarkNeedsReview(chainID)
}

// RunTrace 对实验链执行完整追溯：推进状态→空白传播分析→评分归类→置需复核。
// 同一链串行执行，避免并发裁决竞争。
func (a *App) RunTrace(chainID string) error {
	return a.RunTraceCtx(context.Background(), chainID)
}

// ConfirmPath 确认污染路径（串行裁决）。
func (a *App) ConfirmPath(pathID string) (*model.ContamPath, error) {
	p, err := a.Store.GetPath(pathID)
	if err != nil {
		return nil, err
	}
	mu := a.chainMu(p.ChainID)
	mu.Lock()
	defer mu.Unlock()
	return a.Diag.ConfirmPath(pathID)
}

// RejectPath 否决污染路径（串行裁决）。
func (a *App) RejectPath(pathID string) (*model.ContamPath, error) {
	p, err := a.Store.GetPath(pathID)
	if err != nil {
		return nil, err
	}
	mu := a.chainMu(p.ChainID)
	mu.Lock()
	defer mu.Unlock()
	return a.Diag.RejectPath(pathID)
}

// IsolateBatch 隔离批次并重算（串行裁决）。
func (a *App) IsolateBatch(chainID, batchID string) ([]*model.ContamPath, error) {
	mu := a.chainMu(chainID)
	mu.Lock()
	defer mu.Unlock()
	return a.Diag.IsolateBatch(chainID, batchID)
}

// CreateSnapshot 为实验链创建草稿可信度快照。
func (a *App) CreateSnapshot(chainID string, threshold float64) (*model.Snapshot, error) {
	return a.Snap.CreateDraft(chainID, threshold)
}

// PublishSnapshot 发布快照。
func (a *App) PublishSnapshot(snapshotID string, threshold float64) (*model.Snapshot, error) {
	return a.Snap.Publish(snapshotID, threshold)
}

// Stats 返回各实体计数概览。
func (a *App) Stats() (map[string]int, error) {
	out := map[string]int{}
	samples, err := a.Store.ListSamples()
	if err != nil {
		return nil, err
	}
	out["samples"] = len(samples)
	blanks, err := a.Store.ListBlanks()
	if err != nil {
		return nil, err
	}
	out["blanks"] = len(blanks)
	batches, err := a.Store.ListBatches()
	if err != nil {
		return nil, err
	}
	out["batches"] = len(batches)
	chains, err := a.Store.ListChains()
	if err != nil {
		return nil, err
	}
	out["chains"] = len(chains)
	// 全量路径计数。
	var total int
	for _, c := range chains {
		ps, e := a.Store.ListPathsByChain(c.ID)
		if e != nil {
			continue
		}
		total += len(ps)
	}
	out["paths"] = total
	return out, nil
}
