package snapshot_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
	"task282-ednadiag/internal/snapshot"
)

func newProbeApp(t *testing.T) *service.App {
	t.Helper()
	app, err := service.NewApp(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open app: %v", err)
	}
	t.Cleanup(func() { _ = app.Store.Close() })
	return app
}

func seedContamChain(t *testing.T, app *service.App) string {
	t.Helper()
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

func TestPublishedSnapshotKeepsDraftPayload(t *testing.T) {
	app := newProbeApp(t)
	chainID := seedContamChain(t, app)
	if err := app.RunTrace(chainID); err != nil {
		t.Fatal(err)
	}
	draft, err := app.Snap.CreateDraft(chainID, 0.7)
	if err != nil {
		t.Fatal(err)
	}
	payloadDraft, err := snapshot.ParsePayload(draft)
	if err != nil {
		t.Fatal(err)
	}
	if len(payloadDraft.Taxa) == 0 {
		t.Fatal("empty draft payload")
	}
	paths, _ := app.Store.ListPathsByChain(chainID)
	if len(paths) == 0 {
		t.Fatal("no paths")
	}
	if _, err := app.Diag.RejectPath(paths[0].ID); err != nil {
		t.Fatal(err)
	}
	pub, err := app.Snap.Publish(draft.ID, 0.7)
	if err != nil {
		t.Fatal(err)
	}
	payloadPub, err := snapshot.ParsePayload(pub)
	if err != nil {
		t.Fatal(err)
	}
	if len(payloadPub.Taxa) == 0 || len(payloadDraft.Taxa) == 0 {
		t.Fatal("missing taxa in draft or published payload")
	}
	if payloadPub.Taxa[0].Suspect != payloadDraft.Taxa[0].Suspect {
		t.Fatalf("published suspect drift: draft=%v pub=%v", payloadDraft.Taxa[0].Suspect, payloadPub.Taxa[0].Suspect)
	}
}
