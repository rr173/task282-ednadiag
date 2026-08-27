// Package httpapi 提供基于标准库 net/http 的 HTTP 接口层（路由前缀 /api）。
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"task282-ednadiag/internal/model"
)

// statusClientClosedRequest 表示客户端在响应写入前关闭了连接
// （沿用 nginx 的 499 约定），用于与 5xx 服务端错误区分。
const statusClientClosedRequest = 499

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

// readJSON 解析请求体到 v。
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	return dec.Decode(v)
}

// handleErr 根据错误类型映射状态码。
func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		// 客户端取消或请求超时：不按服务端 5xx 处理。
		writeError(w, statusClientClosedRequest, err)
		return
	}
	if model.IsNotFound(err) {
		notFound(w, err)
		return
	}
	if errors.Is(err, model.ErrSealedImmutable) || errors.Is(err, model.ErrIllegalTransition) {
		writeError(w, http.StatusConflict, err)
		return
	}
	switch err {
	case model.ErrInvalidInput, model.ErrBatchMissing, model.ErrFeatureCodeInvalid,
		model.ErrChainCycle, model.ErrEmptyChain, model.ErrDuplicateID:
		badRequest(w, err)
	default:
		internalErr(w, err)
	}
}
