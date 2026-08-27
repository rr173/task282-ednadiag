package httpapi

import (
	"net/http"

	"task282-ednadiag/internal/model"
)

type createChainReq struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *Handlers) createChain(w http.ResponseWriter, r *http.Request) {
	var req createChainReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	c := &model.Chain{ID: req.ID, Name: req.Name, Status: model.ChainReceiving}
	if err := h.app.Chain.CreateChain(c); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *Handlers) listChains(w http.ResponseWriter, r *http.Request) {
	list, err := h.app.Store.ListChains()
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (h *Handlers) getChain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	c, err := h.app.Store.GetChain(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	steps, err := h.app.Store.ListChainSteps(id)
	if err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chain": c, "steps": steps})
}

type addStepReq struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (h *Handlers) addChainStep(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req addStepReq
	if err := readJSON(r, &req); err != nil {
		badRequest(w, err)
		return
	}
	if err := h.app.Chain.AddStep(id, req.EntityID, req.EntityType); err != nil {
		handleErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"chain_id": id, "entity_id": req.EntityID})
}

func (h *Handlers) traceChain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.app.RunTraceCtx(r.Context(), id); err != nil {
		handleErr(w, err)
		return
	}
	c, _ := h.app.Store.GetChain(id)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handlers) publishChain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.app.Chain.Publish(id); err != nil {
		handleErr(w, err)
		return
	}
	c, _ := h.app.Store.GetChain(id)
	writeJSON(w, http.StatusOK, c)
}

func (h *Handlers) sealChain(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.app.Chain.Seal(id); err != nil {
		handleErr(w, err)
		return
	}
	c, _ := h.app.Store.GetChain(id)
	writeJSON(w, http.StatusOK, c)
}
