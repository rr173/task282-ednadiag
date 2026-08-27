package httpapi

import (
	"net/http"
	"time"

	"task282-ednadiag/internal/model"
)

type createSampleReq struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Site       string `json:"site"`
	Matrix     string `json:"matrix"`
	CollectedAt string `json:"collected_at"`
	Notes      string `json:"notes"`
}

func (h *Handlers) createSample(w http.ResponseWriter, r *http.Request) {
	var req createSampleReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	smp := &model.Sample{
		ID:     req.ID,
		Code:   req.Code,
		Site:   req.Site,
		Matrix: req.Matrix,
		Notes:  req.Notes,
	}
	if req.CollectedAt != "" {
		if t, err := time.Parse(time.RFC3339, req.CollectedAt); err == nil {
			smp.CollectedAt = t
		}
	}
	if err := h.app.Sample.CreateSample(smp); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, smp)
}

func (h *Handlers) listSamples(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.Store.ListSamples()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getSample(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	smp, err := h.app.Store.GetSample(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, smp)
}
