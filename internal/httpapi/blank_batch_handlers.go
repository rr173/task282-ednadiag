package httpapi

import (
	"net/http"
	"time"

	"task282-ednadiag/internal/model"
)

type createBlankReq struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Kind  string `json:"kind"`
	Notes string `json:"notes"`
}

func (h *Handlers) createBlank(w http.ResponseWriter, r *http.Request) {
	var req createBlankReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	b := &model.Blank{ID: req.ID, Code: req.Code, Kind: model.BlankKind(req.Kind), Notes: req.Notes}
	if err := h.app.Sample.CreateBlank(b); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handlers) listBlanks(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.Store.ListBlanks()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getBlank(w http.ResponseWriter, r *http.Request) {
	b, err := h.app.Store.GetBlank(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

type createBatchReq struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Plate string `json:"plate"`
	RunAt string `json:"run_at"`
}

func (h *Handlers) createBatch(w http.ResponseWriter, r *http.Request) {
	var req createBatchReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	b := &model.Batch{ID: req.ID, Code: req.Code, Plate: req.Plate}
	if req.RunAt != "" {
		if t, err := time.Parse(time.RFC3339, req.RunAt); err == nil {
			b.RunAt = t
		}
	}
	if err := h.app.Sample.CreateBatch(b); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handlers) listBatches(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.Store.ListBatches()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getBatch(w http.ResponseWriter, r *http.Request) {
	b, err := h.app.Store.GetBatch(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}
