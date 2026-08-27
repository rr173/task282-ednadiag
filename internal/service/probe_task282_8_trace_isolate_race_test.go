package service_test

import (
	"path/filepath"
	"sync"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

func newProbeApp(t *testing.T) *service.App {
	app, err := service.NewApp(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open app: %v", err)
	}
	t.Cleanup(func() { _ = app.Store.Close() })
	return app
}

func seedContamChain(t *testing.T, app *service.App) string {
	const chainID = "CHAIN-PROBE"
	if err := app.Chain.CreateChain(&model.Chain{ID: chainID, Name: "probe", Status: model.ChainReceiving}); err != nil {
		t.Fatal(err)
	}
	for _, spec := range []struct{ et, eid string }{{"batch", "B1"}, {"blank", "BL1"}, {"sample", "S1"}, {"batch", "B1"}} {
		if err := app.Chain.AddStep(chainID, spec.eid, spec.et); err != nil {
			t.Fatal(err)
		}
	}
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B1", Code: "P1", Plate: "plate"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateBlank(&model.Blank{ID: "BL1", Code: "EB", Kind: model.BlankExtraction}); err != nil {
		t.Fatal(err)
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S1", Code: "SA", Site: "lake", Matrix: "water"}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []*model.Feature{
		{ID: "F-B", Taxon: "carp", Marker: "12S", ReadCount: 10, SourceType: model.SourceBlank, SourceID: "BL1", BatchID: "B1", Quality: 0.9},
		{ID: "F-S", Taxon: "carp", Marker: "12S", ReadCount: 20, SourceType: model.SourceSample, SourceID: "S1", BatchID: "B1", Quality: 0.8},
	} {
		if err := app.Sample.CreateFeature(f); err != nil {
			t.Fatal(err)
		}
	}
	return chainID
}

func TestConcurrentTraceIsolateBatchConsistent(t *testing.T) {
	app := newProbeApp(t)
	chainID := seedContamChain(t, app)
	const workers = 20
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = app.RunTrace(chainID)
				return
			}
			_, _ = app.IsolateBatch(chainID, "B1")
		}(i)
	}
	wg.Wait()
	b, err := app.Store.GetBatch("B1")
	if err != nil {
		t.Fatal(err)
	}
	if !b.Isolated {
		t.Fatal("expected batch B1 isolated after concurrent isolate attempts")
	}
	paths, err := app.Store.ListPathsByChain(chainID)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p.ViaBatchID == "B1" {
			t.Fatalf("isolated batch B1 still has propagation paths: %+v", p)
		}
	}
}
