package service_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

// seedChain 构造一条可被追溯的实验链（carp 同时出现在空白与同批次样本中，
// 触发污染路径）。
func seedChain(t *testing.T, app *service.App) {
	t.Helper()
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
	if err := app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "cancel-chain"}); err != nil {
		t.Fatal(err)
	}
	for _, e := range []struct{ typ, id string }{
		{"sample", "S1"}, {"blank", "BL1"}, {"batch", "B1"},
	} {
		if err := app.Chain.AddStep("C1", e.id, e.typ); err != nil {
			t.Fatal(err)
		}
	}
}

// TestRunTraceCtxRespectsCancellation 验证客户端取消后不再执行后续阶段：
// 进入锁后 ctx 已取消，RunTraceCtx 应立即返回 context.Canceled，链状态不推进
// （停在接收态，Trace 未执行），路径候选未生成。
func TestRunTraceCtxRespectsCancellation(t *testing.T) {
	app, err := service.NewApp(filepath.Join(t.TempDir(), "cancel.db"))
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	seedChain(t, app)

	// 提前取消 ctx：进入锁后第一个阶段间检查即命中，后续四阶段均不执行。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := app.RunTraceCtx(ctx, "C1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	c, err := app.Store.GetChain("C1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != model.ChainReceiving {
		t.Fatalf("expected status %q (Trace 未执行), got %q", model.ChainReceiving, c.Status)
	}
	paths, err := app.Store.ListPathsByChain("C1")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 0 {
		t.Fatalf("expected no paths after cancellation, got %d", len(paths))
	}
}

// TestRunTraceCtxCompletesWhenNotCancelled 验证未取消时整条追溯正常完成，
// 作为取消修复的对照基线。
func TestRunTraceCtxCompletesWhenNotCancelled(t *testing.T) {
	app, err := service.NewApp(filepath.Join(t.TempDir(), "complete.db"))
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	seedChain(t, app)

	if err := app.RunTraceCtx(context.Background(), "C1"); err != nil {
		t.Fatalf("RunTraceCtx: %v", err)
	}
	c, err := app.Store.GetChain("C1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Status != model.ChainNeedsReview {
		t.Fatalf("expected status %q, got %q", model.ChainNeedsReview, c.Status)
	}
	if paths, _ := app.Store.ListPathsByChain("C1"); len(paths) == 0 {
		t.Fatal("expected contamination paths after full trace")
	}
}

// TestRunTraceCtxSerialAfterCancellation 验证取消路径正确释放链锁：
// 一次已取消的调用返回后，同链的下一次正常调用必须能获取锁并完成。
func TestRunTraceCtxSerialAfterCancellation(t *testing.T) {
	app, err := service.NewApp(filepath.Join(t.TempDir(), "serial.db"))
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	seedChain(t, app)

	ctxCancel, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.RunTraceCtx(ctxCancel, "C1"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled call: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- app.RunTraceCtx(context.Background(), "C1") }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("call after cancellation should succeed, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunTraceCtx did not complete: lock likely leaked from cancellation path")
	}

	c, _ := app.Store.GetChain("C1")
	if c.Status != model.ChainNeedsReview {
		t.Fatalf("expected status %q, got %q", model.ChainNeedsReview, c.Status)
	}
}
