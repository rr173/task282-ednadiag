package store

import (
	"path/filepath"
	"testing"

	"task282-ednadiag/internal/model"
)

// TestPublishSnapshotNotFoundRollsBack 验证发布不存在的快照时事务被回滚，
// 不会占住单连接导致后续写入卡住或报 database locked。
func TestPublishSnapshotNotFoundRollsBack(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pubsnap.db")
	s, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	// 插入一个草稿快照供后续发布验证。
	snap := &model.Snapshot{ID: "snap1", ChainID: "C1", Status: model.SnapDraft, Threshold: 0.5}
	if err := s.CreateSnapshot(snap); err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	// 失败路径：发布不存在的快照应返回 ErrNotFound 并回滚事务。
	if err := s.PublishSnapshot("does-not-exist", 0.5); err == nil {
		t.Fatal("expected error publishing nonexistent snapshot, got nil")
	}

	// 关键回归点：单连接被悬挂事务占住后，后续写入会卡住或报 database locked。
	// 这里用正常发布验证连接仍可用，并立即以另一写入验证未被锁住。
	if err := s.PublishSnapshot("snap1", 0.5); err != nil {
		t.Fatalf("subsequent publish after failed path: %v (transaction was not rolled back)", err)
	}
	got, err := s.GetSnapshot("snap1")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if got.Status != model.SnapPublished {
		t.Fatalf("snapshot status = %s, want published", got.Status)
	}

	// 再次用一条全新写入验证连接确实空闲（非 busy、非 locked）。
	snap2 := &model.Snapshot{ID: "snap2", ChainID: "C1", Status: model.SnapDraft, Threshold: 0.5}
	if err := s.CreateSnapshot(snap2); err != nil {
		t.Fatalf("subsequent write locked: %v", err)
	}
}
