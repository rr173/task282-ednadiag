package service_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

// TestIsolateBatchCutsPropagation 隔离批次后重新分析，该批次的传播路径不应再生成。
// 复现并守护：隔离标记必须参与每次 Analyze（曾因包级缓存导致隔离失效）。
func TestIsolateBatchCutsPropagation(t *testing.T) {
	db := filepath.Join(t.TempDir(), "iso.db")
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()

	// 批次 B1 含空白与同批次样本（应产生 carp 污染路径）。
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B1", Code: "PLT1", Plate: "P1"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S1", Code: "FLT-A", Site: "lake", Matrix: "water"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateBlank(&model.Blank{ID: "BL1", Code: "EB-1", Kind: model.BlankExtraction}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F1", Taxon: "carp", Marker: "12S", ReadCount: 120, SourceType: model.SourceSample, SourceID: "S1", BatchID: "B1", Quality: 0.9}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F2", Taxon: "carp", Marker: "12S", ReadCount: 40, SourceType: model.SourceBlank, SourceID: "BL1", BatchID: "B1", Quality: 0.8}); err != nil {
		t.Fatal(err)
	}

	if err := app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "iso-chain"}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct{ typ, id string }{{"sample", "S1"}, {"blank", "BL1"}, {"batch", "B1"}} {
		if err := app.Chain.AddStep("C1", e.id, e.typ); err != nil {
			t.Fatal(err)
		}
	}

	// 首次追溯：应生成 carp 经 B1 传播的路径。
	if err := app.RunTrace("C1"); err != nil {
		t.Fatalf("trace: %v", err)
	}
	before := pathsViaBatch(app, "C1", "B1")
	if len(before) == 0 {
		t.Fatalf("expected contamination path via B1 before isolation, got none")
	}

	// 隔离 B1 并重算：路径应被切断，不再生成经 B1 的传播路径。
	if _, err := app.IsolateBatch("C1", "B1"); err != nil {
		t.Fatalf("isolate: %v", err)
	}
	after := pathsViaBatch(app, "C1", "B1")
	if len(after) != 0 {
		t.Fatalf("expected no propagation path via isolated B1, got %d: %+v", len(after), after)
	}
}

func pathsViaBatch(app *service.App, chainID, batchID string) []*model.ContamPath {
	all, err := app.Store.ListPathsByChain(chainID)
	if err != nil {
		return nil
	}
	var out []*model.ContamPath
	for _, p := range all {
		if p.ViaBatchID == batchID {
			out = append(out, p)
		}
	}
	return out
}
