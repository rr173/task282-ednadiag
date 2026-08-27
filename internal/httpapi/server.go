package httpapi

import (
	"net/http"

	"task282-ednadiag/internal/service"
)

// Handlers 聚合全部 HTTP 处理方法，持有应用编排根。
type Handlers struct {
	app *service.App
}

// NewServer 构造挂载全部 /api 路由的 http.Handler。
func NewServer(app *service.App) http.Handler {
	h := &Handlers{app: app}
	mux := http.NewServeMux()

	// 样本
	mux.HandleFunc("POST /api/samples", h.createSample)
	mux.HandleFunc("GET /api/samples", h.listSamples)
	mux.HandleFunc("GET /api/samples/{id}", h.getSample)

	// 空白
	mux.HandleFunc("POST /api/blanks", h.createBlank)
	mux.HandleFunc("GET /api/blanks", h.listBlanks)
	mux.HandleFunc("GET /api/blanks/{id}", h.getBlank)

	// 批次
	mux.HandleFunc("POST /api/batches", h.createBatch)
	mux.HandleFunc("GET /api/batches", h.listBatches)
	mux.HandleFunc("GET /api/batches/{id}", h.getBatch)

	// 序列特征
	mux.HandleFunc("POST /api/features", h.createFeature)
	mux.HandleFunc("GET /api/features", h.listFeatures)
	mux.HandleFunc("GET /api/features/{id}", h.getFeature)

	// 实验链
	mux.HandleFunc("POST /api/chains", h.createChain)
	mux.HandleFunc("GET /api/chains", h.listChains)
	mux.HandleFunc("GET /api/chains/{id}", h.getChain)
	mux.HandleFunc("POST /api/chains/{id}/steps", h.addChainStep)
	mux.HandleFunc("POST /api/chains/{id}/trace", h.traceChain)
	mux.HandleFunc("POST /api/chains/{id}/publish", h.publishChain)
	mux.HandleFunc("POST /api/chains/{id}/seal", h.sealChain)

	// 污染路径与诊断
	mux.HandleFunc("GET /api/chains/{id}/paths", h.listPaths)
	mux.HandleFunc("POST /api/paths/{id}/confirm", h.confirmPath)
	mux.HandleFunc("POST /api/paths/{id}/reject", h.rejectPath)
	mux.HandleFunc("POST /api/chains/{id}/isolate", h.isolateBatch)

	// 可信度快照
	mux.HandleFunc("POST /api/snapshots", h.createSnapshot)
	mux.HandleFunc("POST /api/snapshots/{id}/publish", h.publishSnapshot)
	mux.HandleFunc("GET /api/snapshots/{id}", h.getSnapshot)

	// 自检
	mux.HandleFunc("GET /api/stats", h.stats)
	mux.HandleFunc("GET /api/selfcheck", h.selfCheck)

	return mux
}
