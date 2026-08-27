package main_test

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/service"
)

func TestNewAppOpensDatabase(t *testing.T) {
	db := filepath.Join(t.TempDir(), "unit.db")
	app, err := service.NewApp(db)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	defer app.Store.Close()
	if err := app.Sample.CreateBatch(&model.Batch{ID: "TB", Code: "c", Plate: "p"}); err != nil {
		t.Fatal(err)
	}
	b, err := app.Store.GetBatch("TB")
	if err != nil || b.Code != "c" {
		t.Fatalf("batch: %v %v", b, err)
	}
}
