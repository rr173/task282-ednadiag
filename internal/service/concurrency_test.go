package service

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"task282-ednadiag/internal/model"
)

// seedContamChain 录入一条“空白+样本同批次共现”的实验链，追溯后应稳定产生 1 条污染路径。
func seedContamChain(t *testing.T, app *App) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(app.Sample.CreateBatch(&model.Batch{ID: "B1", Code: "PLT1", Plate: "P1"}))
	must(app.Sample.CreateSample(&model.Sample{ID: "S1", Code: "FLT-A", Site: "lake", Matrix: "water"}))
	must(app.Sample.CreateBlank(&model.Blank{ID: "BL1", Code: "EB-1", Kind: model.BlankExtraction}))
	must(app.Sample.CreateFeature(&model.Feature{ID: "F1", Taxon: "carp", Marker: "12S", ReadCount: 120, SourceType: model.SourceSample, SourceID: "S1", BatchID: "B1", Quality: 0.9}))
	must(app.Sample.CreateFeature(&model.Feature{ID: "F2", Taxon: "carp", Marker: "12S", ReadCount: 40, SourceType: model.SourceBlank, SourceID: "BL1", BatchID: "B1", Quality: 0.8}))
	must(app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "concurrency-chain"}))
	for _, e := range []struct{ typ, id string }{{"sample", "S1"}, {"blank", "BL1"}, {"batch", "B1"}} {
		must(app.Chain.AddStep("C1", e.id, e.typ))
	}
}

// TestConcurrentTraceStablePaths 模拟 20 位同事同时对同一实验链触发追溯，
// 并在追溯进行期间不断读取路径列表：任何一次读取都必须非空且条数稳定。
func TestConcurrentTraceStablePaths(t *testing.T) {
	db := filepath.Join(t.TempDir(), "concurrent.db")
	app, err := NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	seedContamChain(t, app)

	// 先跑一次追溯，使链上已有 1 条污染路径；此后任何并发读都不应再看到空列表或条数漂移。
	if err := app.RunTraceCtx(context.Background(), "C1"); err != nil {
		t.Fatalf("seed trace: %v", err)
	}

	const colleagues = 20
	var wg sync.WaitGroup
	var traceErr error
	var errMu sync.Mutex
	recordErr := func(e error) {
		errMu.Lock()
		if e != nil && traceErr == nil {
			traceErr = e
		}
		errMu.Unlock()
	}

	// 20 位同事并发触发同一链的追溯。
	for i := 0; i < colleagues; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			recordErr(app.RunTraceCtx(context.Background(), "C1"))
		}()
	}

	// 读取端：在追溯推进期间反复读路径，断言“非空且条数恒定（始终为 1）”。
	// 修复前，读端会落在 DeletePathsByChain 之后、CreatePath 之前的空窗，看到空列表或漂移。
	stop := make(chan struct{})
	var readFail error
	var failMu sync.Mutex
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
			}
			ps, e := app.ListPaths("C1")
			if e != nil {
				failMu.Lock()
				if readFail == nil {
					readFail = e
				}
				failMu.Unlock()
				return
			}
			// 追溯必然产出 1 条 carp 路径；并发读不得看到空列表或条数漂移。
			if len(ps) != 1 {
				failMu.Lock()
				if readFail == nil {
					readFail = errUnstablePathList(len(ps))
				}
				failMu.Unlock()
				return
			}
		}
	}()

	wg.Wait()
	close(stop)
	if traceErr != nil {
		t.Fatalf("concurrent trace failed: %v", traceErr)
	}
	if readFail != nil {
		t.Fatalf("concurrent read saw unstable/empty path list: %v", readFail)
	}

	// 最终落定后再次校验：条数与内容稳定。
	ps, err := app.ListPaths("C1")
	if err != nil {
		t.Fatalf("final ListPaths: %v", err)
	}
	if len(ps) != 1 || ps[0].Taxon != "carp" {
		t.Fatalf("expected exactly 1 carp path, got %#v", ps)
	}
}

// TestListPathsUnknownChain404 保证未知链返回 ErrNotFound（映射 HTTP 404）而非空列表。
func TestListPathsUnknownChain404(t *testing.T) {
	db := filepath.Join(t.TempDir(), "unknown.db")
	app, err := NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	if _, err := app.ListPaths("nope"); err != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

type errStr string

func (e errStr) Error() string { return string(e) }

func errUnstablePathList(n int) error {
	return errStr(fmt.Sprintf("path list unstable during concurrent trace: got %d paths (expected 1)", n))
}
