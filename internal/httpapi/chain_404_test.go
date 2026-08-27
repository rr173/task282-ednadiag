package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"task282-ednadiag/internal/service"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	db := filepath.Join(t.TempDir(), "unit.db")
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	t.Cleanup(func() { app.Store.Close() })
	return NewServer(app)
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestGetNonexistentChainReturns404(t *testing.T) {
	h := newTestServer(t)

	w := do(t, h, http.MethodGet, "/api/chains/nope", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /api/chains/nope: want 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestTraceNonexistentChainReturns404(t *testing.T) {
	h := newTestServer(t)

	w := do(t, h, http.MethodPost, "/api/chains/nope/trace", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("POST /api/chains/nope/trace: want 404, got %d (body=%s)", w.Code, w.Body.String())
	}
}
