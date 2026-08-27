// Package httpapi 提供基于标准库 net/http 的 HTTP 接口层（路由前缀 /api）。
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"task282-ednadiag/internal/model"
)

// writeJSON 写 JSON 响应。
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError 写错误响应（含错误码与消息）。
func writeError(w http.ResponseWriter, code int, err error) {
	msg := err.Error()
	writeJSON(w, code, map[string]string{"error": msg})
}

// badRequest 写 400。
func badRequest(w http.ResponseWriter, err error) { writeError(w, http.StatusBadRequest, err) }

// notFound 写 404。
func notFound(w http.ResponseWriter, err error) { writeError(w, http.StatusNotFound, err) }

// internalErr 写 500。
func internalErr(w http.ResponseWriter, err error) { writeError(w, http.StatusInternalServerError, err) }

// conflict 写 409（冲突：封存不可变、非法状态迁移等）。
func conflict(w http.ResponseWriter, err error) { writeError(w, http.StatusConflict, err) }

// readJSON 解析请求体到 v。
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

// handleErr 根据错误类型映射状态码。
func handleErr(w http.ResponseWriter, err error) {
	if model.IsNotFound(err) {
		notFound(w, err)
		return
	}
	switch {
	case errors.Is(err, model.ErrSealedImmutable) || errors.Is(err, model.ErrIllegalTransition):
		conflict(w, err)
	case errors.Is(err, model.ErrInvalidInput) || errors.Is(err, model.ErrBatchMissing) ||
		errors.Is(err, model.ErrFeatureCodeInvalid) || errors.Is(err, model.ErrChainCycle) ||
		errors.Is(err, model.ErrEmptyChain) || errors.Is(err, model.ErrDuplicateID):
		badRequest(w, err)
	default:
		internalErr(w, err)
	}
}
