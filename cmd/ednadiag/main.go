// Command ednadiag 是环境 DNA 采样空白传播诊断服务的入口：
// 默认启动 HTTP 服务；指定 --smoke-test 时执行端到端自检（含数据库关闭重开的持久化验证）。
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"task282-ednadiag/internal/httpapi"
	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
	"task282-ednadiag/internal/snapshot"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP 监听地址")
	dbPath := flag.String("db", "ednadiag.db", "SQLite 数据库路径")
	smoke := flag.Bool("smoke-test", false, "执行端到端自检后退出（不启动长驻服务）")
	flag.Parse()

	if *smoke {
		if err := runSmokeTest(*dbPath); err != nil {
			log.Fatalf("smoke-test failed: %v", err)
		}
		fmt.Println("smoke-test PASSED")
		return
	}

	app, err := service.NewApp(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer app.Store.Close()

	srv := &http.Server{
		Addr:              *addr,
		Handler:           httpapi.NewServer(app),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("ednadiag listening on %s (db=%s)", *addr, *dbPath)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runSmokeTest 端到端自检：构造实验链与空白污染场景，运行追溯与快照，
// 关闭数据库后重新打开验证持久化与重启恢复。
func runSmokeTest(dbPath string) error {
	if dbPath == "" || dbPath == "ednadiag.db" {
		dbPath = filepath.Join(os.TempDir(), fmt.Sprintf("ednadiag_smoke_%d.db", time.Now().UnixNano()))
	}
	_ = os.Remove(dbPath)

	app, err := service.NewApp(dbPath)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}

	// 录入输入：两个批次，一个含空白、一个不含。
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B1", Code: "PLT1", Plate: "P1"}); err != nil {
		return err
	}
	if err := app.Sample.CreateBatch(&model.Batch{ID: "B2", Code: "PLT2", Plate: "P2"}); err != nil {
		return err
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S1", Code: "FLT-A", Site: "lake", Matrix: "water"}); err != nil {
		return err
	}
	if err := app.Sample.CreateSample(&model.Sample{ID: "S2", Code: "FLT-B", Site: "river", Matrix: "water"}); err != nil {
		return err
	}
	if err := app.Sample.CreateBlank(&model.Blank{ID: "BL1", Code: "EB-1", Kind: model.BlankExtraction}); err != nil {
		return err
	}

	// 序列特征：carp 同时出现在空白与同批次样本（应判污染）；trout 仅出现在无空白批次（可信）。
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F1", Taxon: "carp", Marker: "12S", ReadCount: 120, SourceType: model.SourceSample, SourceID: "S1", BatchID: "B1", Quality: 0.9}); err != nil {
		return err
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F2", Taxon: "carp", Marker: "12S", ReadCount: 40, SourceType: model.SourceBlank, SourceID: "BL1", BatchID: "B1", Quality: 0.8}); err != nil {
		return err
	}
	if err := app.Sample.CreateFeature(&model.Feature{ID: "F3", Taxon: "trout", Marker: "12S", ReadCount: 200, SourceType: model.SourceSample, SourceID: "S2", BatchID: "B2", Quality: 0.95}); err != nil {
		return err
	}

	// 构建实验链并关联步骤。
	if err := app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "demo-chain"}); err != nil {
		return err
	}
	for _, e := range []struct {
		typ, id string
	}{{"sample", "S1"}, {"blank", "BL1"}, {"batch", "B1"}, {"sample", "S2"}, {"batch", "B2"}} {
		if err := app.Chain.AddStep("C1", e.id, e.typ); err != nil {
			return err
		}
	}

	// 运行追溯（传播分析 + 评分）。
	if err := app.RunTrace("C1"); err != nil {
		return fmt.Errorf("trace: %w", err)
	}

	// 校验生成了 carp 的污染路径且被判为污染确认。
	paths, err := app.Store.ListPathsByChain("C1")
	if err != nil {
		return err
	}
	foundCarp := false
	for _, p := range paths {
		if p.Taxon == "carp" && p.Status == model.PathConfirmed {
			foundCarp = true
		}
	}
	if !foundCarp {
		return fmt.Errorf("expected confirmed contamination path for carp, got %d paths", len(paths))
	}

	// 创建并发布可信度快照。
	snap, err := app.CreateSnapshot("C1", 0.5)
	if err != nil {
		return err
	}
	if _, err := app.PublishSnapshot(snap.ID, 0.5); err != nil {
		return err
	}

	// 关闭数据库，模拟重启。
	if err := app.Store.Close(); err != nil {
		return err
	}

	// 重新打开并验证持久化与恢复。
	app2, err := service.NewApp(dbPath)
	if err != nil {
		return fmt.Errorf("reopen: %w", err)
	}
	defer app2.Store.Close()

	c, err := app2.Store.GetChain("C1")
	if err != nil {
		return fmt.Errorf("reopen chain: %w", err)
	}
	if c.Status != model.ChainNeedsReview {
		return fmt.Errorf("chain status not persisted: %s", c.Status)
	}
	paths2, err := app2.Store.ListPathsByChain("C1")
	if err != nil {
		return err
	}
	if len(paths2) != len(paths) {
		return fmt.Errorf("paths not persisted: before=%d after=%d", len(paths), len(paths2))
	}
	snap2, err := app2.Store.GetSnapshot(snap.ID)
	if err != nil {
		return fmt.Errorf("reopen snapshot: %w", err)
	}
	if snap2.Status != model.SnapPublished {
		return fmt.Errorf("snapshot status not persisted: %s", snap2.Status)
	}
	payload, err := snapshot.ParsePayload(snap2)
	if err != nil {
		return err
	}
	fmt.Print(snapshot.Summary(payload))
	log.Printf("restart recovery OK: chain=%s paths=%d snapshot=%s", c.Status, len(paths2), snap2.Status)
	return nil
}
