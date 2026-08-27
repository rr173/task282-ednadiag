package service_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/service"
)

func newProbeApp(t *testing.T) *service.App {
	app, err := service.NewApp(filepath.Join(t.TempDir(), "probe.db"))
	if err != nil {
		t.Fatalf("open app: %v", err)
	}
	t.Cleanup(func() { _ = app.Store.Close() })
	return app
}

func TestConfirmMissingPathReturnsError(t *testing.T) {
	app := newProbeApp(t)
	if _, err := app.ConfirmPath("missing-path-id"); err == nil {
		t.Fatal("expected error confirming missing path")
	}
}
