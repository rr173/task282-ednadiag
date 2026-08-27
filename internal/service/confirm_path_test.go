package service_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

// TestConfirmRejectMissingPathReturnsError 锁定回归：确认/否决不存在的污染路径时
// 必须返回明确错误（ErrNotFound），而非吞掉错误返回 (nil, nil)，
// 否则 HTTP 层会向客户端回 200 空 body（缺失路径未返回明确错误）。
func TestConfirmRejectMissingPathReturnsError(t *testing.T) {
	db := filepath.Join(t.TempDir(), "unit.db")
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()

	for _, fn := range []struct {
		name string
		call func(string) (*model.ContamPath, error)
	}{
		{"ConfirmPath", app.ConfirmPath},
		{"RejectPath", app.RejectPath},
	} {
		t.Run(fn.name, func(t *testing.T) {
			p, err := fn.call("does-not-exist")
			if err == nil {
				t.Fatalf("%s: expected error for missing path, got nil (path=%v)", fn.name, p)
			}
			if !model.IsNotFound(err) {
				t.Fatalf("%s: expected ErrNotFound, got %v", fn.name, err)
			}
			if p != nil {
				t.Fatalf("%s: expected nil path on error, got %v", fn.name, p)
			}
		})
	}
}
