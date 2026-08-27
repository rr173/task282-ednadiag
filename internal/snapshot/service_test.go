package snapshot_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
	"task282-ednadiag/internal/snapshot"
	"task282-ednadiag/internal/store"
)

// TestPublishFreezesDraftPayload 验证发布只固化草稿载荷：
// 发布后否决某路径再发布同一快照，已发布快照里的分类单元条目不应跟着 live 路径变。
func TestPublishFreezesDraftPayload(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "snap.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	app := service.NewAppWithStore(st)
	defer app.Store.Close()

	// 两个批次：B1 含空白，B2 不含。
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B1", Code: "PLT1", Plate: "P1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B2", Code: "PLT2", Plate: "P2"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S1", Code: "FLT-A", Site: "lake", Matrix: "water"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S2", Code: "FLT-B", Site: "river", Matrix: "water"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateBlank(&model.Blank{ID: "BL1", Code: "EB-1", Kind: model.BlankExtraction}); err != nil {
		t.Fatal(err)
	}

	// carp 同时出现在空白与同批次样本（判污染）；trout 仅出现在无空白批次（可信）。
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F1", Taxon: "carp", Marker: "12S", ReadCount: 120, SourceType: model.SourceSample, SourceID: "S1", BatchID: "B1", Quality: 0.9}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F2", Taxon: "carp", Marker: "12S", ReadCount: 40, SourceType: model.SourceBlank, SourceID: "BL1", BatchID: "B1", Quality: 0.8}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F3", Taxon: "trout", Marker: "12S", ReadCount: 200, SourceType: model.SourceSample, SourceID: "S2", BatchID: "B2", Quality: 0.95}); err != nil {
		t.Fatal(err)
	}

	if err := app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct{ typ, id string }{
		{"sample", "S1"}, {"blank", "BL1"}, {"batch", "B1"},
		{"sample", "S2"}, {"batch", "B2"},
	} {
		if err := app.Chain.AddStep("C1", e.id, e.typ); err != nil {
			t.Fatal(err)
		}
	}

	// 追溯：传播分析 + 评分归类。carp 应被判为污染确认（score 0.85）。
	if err := app.RunTrace("C1"); err != nil {
		t.Fatalf("trace: %v", err)
	}
	var carpPathID string
	paths, err := app.Store.ListPathsByChain("C1")
	if err != nil {
		t.Fatalf("list paths: %v", err)
	}
	for _, p := range paths {
		if p.Taxon == "carp" {
			carpPathID = p.ID
			if p.Status != model.PathConfirmed {
				t.Fatalf("carp path expected confirmed, got %s", p.Status)
			}
		}
	}
	if carpPathID == "" {
		t.Fatal("no carp path after trace")
	}

	// 创建并发布快照：载荷含 carp，suspect=true。
	snap, err := app.CreateSnapshot("C1", 0.5)
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	published, err := app.PublishSnapshot(snap.ID, 0.5)
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if published.Status != model.SnapPublished {
		t.Fatalf("status=%s want published", published.Status)
	}
	payloadBefore := parseSnap(t, published)
	carpBefore := findTaxon(t, payloadBefore, "carp")
	if carpBefore == nil {
		t.Fatalf("published payload missing carp: %+v", payloadBefore.Taxa)
	}
	if !carpBefore.Suspect {
		t.Fatalf("carp not flagged suspect: %+v", carpBefore)
	}

	// 发布后否决 carp 路径（live 状态变为 rejected）。
	if _, err := app.Diag.RejectPath(carpPathID); err != nil {
		t.Fatalf("reject carp: %v", err)
	}

	// 再次发布同一快照：已固化的载荷不应被按 live 路径重建，carp 条目必须保留不变。
	republished, err := app.PublishSnapshot(snap.ID, 0.5)
	if err != nil {
		t.Fatalf("republish: %v", err)
	}
	payloadAfter := parseSnap(t, republished)
	carpAfter := findTaxon(t, payloadAfter, "carp")
	if carpAfter == nil {
		t.Fatalf("republished payload dropped carp after rejection (rebuild leak): %+v", payloadAfter.Taxa)
	}
	if carpAfter.Suspect != carpBefore.Suspect || carpAfter.Credibility != carpBefore.Credibility {
		t.Fatalf("carp changed across republish: before=%+v after=%+v", carpBefore, carpAfter)
	}
}

func parseSnap(t *testing.T, snap *model.Snapshot) model.SnapshotPayload {
	t.Helper()
	p, err := snapshot.ParsePayload(snap)
	if err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	return p
}

func findTaxon(t *testing.T, p model.SnapshotPayload, taxon string) *model.TaxonCredibility {
	t.Helper()
	for i := range p.Taxa {
		if p.Taxa[i].Taxon == taxon {
			return &p.Taxa[i]
		}
	}
	return nil
}
