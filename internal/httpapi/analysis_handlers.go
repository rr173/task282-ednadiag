package httpapi

import (
	"net/http"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/snapshot"
)

func (h *Handlers) listPaths(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	paths, err := h.app.ListPaths(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

func (h *Handlers) confirmPath(w http.ResponseWriter, r *http.Request) {
	p, err := h.app.ConfirmPath(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handlers) rejectPath(w http.ResponseWriter, r *http.Request) {
	p, err := h.app.RejectPath(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

type isolateReq struct {
	BatchID string `json:"batch_id"`
}

func (h *Handlers) isolateBatch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req isolateReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	if req.BatchID == "" {
		badRequest(w, model.ErrInvalidInput)
		return
	}
	paths, err := h.app.IsolateBatch(id, req.BatchID)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, paths)
}

type createSnapReq struct {
	ChainID  string  `json:"chain_id"`
	Threshold float64 `json:"threshold"`
}

func (h *Handlers) createSnapshot(w http.ResponseWriter, r *http.Request) {
	var req createSnapReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	snap, err := h.app.CreateSnapshot(req.ChainID, req.Threshold)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, snap)
}

type publishSnapReq struct {
	Threshold float64 `json:"threshold"`
}

func (h *Handlers) publishSnapshot(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req publishSnapReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	snap, err := h.app.PublishSnapshot(id, req.Threshold)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

func (h *Handlers) getSnapshot(w http.ResponseWriter, r *http.Request) {
	snap, err := h.app.Store.GetSnapshot(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	payload, err := snapshot.ParsePayload(snap)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshot": snap, "payload": payload})
}

func (h *Handlers) stats(w http.ResponseWriter, r *http.Request) {
	s, err := h.app.Stats()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handlers) selfCheck(w http.ResponseWriter, r *http.Request) {
	// 验证数据库连通。
	if _, err := h.app.Store.ListChains(); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
