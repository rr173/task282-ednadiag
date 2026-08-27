package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

// post 调用路由并返回响应状态码与响应体。
func doRequest(t *testing.T, mux http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// TestTraceSealedChainReturnsConflict 验证对已封存实验链再次追溯返回 409 而非 500。
// 封存链不可变，ErrSealedImmutable 必须映射为冲突状态码。
func TestTraceSealedChainReturnsConflict(t *testing.T) {
	db := ":memory:"
	// 内存库由 NewApp 构造；这里复用临时文件以保持与现代 sqlite 驱动一致的用法。
	db = t.TempDir() + "/sealed.db"
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()

	// 创建实验链并直接置为封存态（绕过完整状态机，聚焦错误码映射）。
	if err := app.Chain.CreateChain(&model.Chain{ID: "C1", Name: "sealed-chain"}); err != nil {
		t.Fatalf("create chain: %v", err)
	}
	if err := app.Store.UpdateChainStatus("C1", model.ChainSealed); err != nil {
		t.Fatalf("set sealed: %v", err)
	}

	mux := NewServer(app)

	// 对已封存链再次追溯：应返回 409，而非 500。
	w := doRequest(t, mux, http.MethodPost, "/api/chains/C1/trace", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("trace sealed chain: want status %d (Conflict), got %d; body=%s",
			http.StatusConflict, w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), model.ErrSealedImmutable.Error()) {
		t.Fatalf("trace sealed chain: want error message containing %q, got %s",
			model.ErrSealedImmutable.Error(), w.Body.String())
	}
}

// TestSealSealedChainReturnsConflict 验证对已封存链再次封存同样返回 409。
func TestSealSealedChainReturnsConflict(t *testing.T) {
	db := t.TempDir() + "/sealed2.db"
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()

	if err := app.Chain.CreateChain(&model.Chain{ID: "C2", Name: "sealed-chain-2"}); err != nil {
		t.Fatalf("create chain: %v", err)
	}
	if err := app.Store.UpdateChainStatus("C2", model.ChainSealed); err != nil {
		t.Fatalf("set sealed: %v", err)
	}

	mux := NewServer(app)
	w := doRequest(t, mux, http.MethodPost, "/api/chains/C2/seal", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("seal sealed chain: want status %d (Conflict), got %d; body=%s",
			http.StatusConflict, w.Code, w.Body.String())
	}
}

// TestPublishSealedChainReturnsConflict 验证对已封存链发布同样返回 409。
func TestPublishSealedChainReturnsConflict(t *testing.T) {
	db := t.TempDir() + "/sealed3.db"
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()

	if err := app.Chain.CreateChain(&model.Chain{ID: "C3", Name: "sealed-chain-3"}); err != nil {
		t.Fatalf("create chain: %v", err)
	}
	if err := app.Store.UpdateChainStatus("C3", model.ChainSealed); err != nil {
		t.Fatalf("set sealed: %v", err)
	}

	mux := NewServer(app)
	w := doRequest(t, mux, http.MethodPost, "/api/chains/C3/publish", "")
	if w.Code != http.StatusConflict {
		t.Fatalf("publish sealed chain: want status %d (Conflict), got %d; body=%s",
			http.StatusConflict, w.Code, w.Body.String())
	}
}
