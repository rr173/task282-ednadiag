package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"task282-ednadiag/internal/model"
	"task282-ednadiag/internal/store"
)

func TestPublishSnapshotMissingIDDoesNotLockDB(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "p.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.CreateSnapshot(&model.Snapshot{ID: "s1", ChainID: "c", Status: model.SnapDraft, Payload: "{}"}); err != nil {
		t.Fatal(err)
	}
	if err := st.PublishSnapshot("missing", 0.7); err == nil {
		t.Fatal("expected error")
	}
	done := make(chan error, 1)
	go func() {
		done <- st.CreateSnapshot(&model.Snapshot{ID: "s2", ChainID: "c", Status: model.SnapDraft, Payload: "{}"})
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("db locked after failed publish: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("db locked after failed publish: timed out waiting for CreateSnapshot")
	}
}
