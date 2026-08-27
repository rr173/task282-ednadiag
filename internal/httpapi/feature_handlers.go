package httpapi

import (
	"net/http"

	"task282-ednadiag/internal/model"
)

type createFeatureReq struct {
	ID         string  `json:"id"`
	Taxon      string  `json:"taxon"`
	Marker     string  `json:"marker"`
	ReadCount  int     `json:"read_count"`
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	BatchID    string  `json:"batch_id"`
	Quality    float64 `json:"quality"`
}

func (h *Handlers) createFeature(w http.ResponseWriter, r *http.Request) {
	var req createFeatureReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	f := &model.Feature{
		ID:         req.ID,
		Taxon:      req.Taxon,
		Marker:     req.Marker,
		ReadCount:  req.ReadCount,
		SourceType: model.SourceType(req.SourceType),
		SourceID:   req.SourceID,
		BatchID:    req.BatchID,
		Quality:    req.Quality,
		Status:     model.FeatureRaw,
	}
	if err := h.app.Sample.CreateFeature(f); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handlers) listFeatures(w http.ResponseWriter, r *http.Request) {
	taxon := r.URL.Query().Get("taxon")
	src := r.URL.Query().Get("source_type")
	list, err := h.app.Store.ListFeatures(taxon, src)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getFeature(w http.ResponseWriter, r *http.Request) {
	f, err := h.app.Store.GetFeature(r.PathValue("id"))
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}
