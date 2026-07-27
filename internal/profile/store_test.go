package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Save()가 실패 경로에서 임시 파일을 지우는지(D11)는 add 커맨드 전체로는 확인할 수 없다 —
// 목적지 자리에 미리 디렉토리를 놓으면 Load()가 그 자리에서 먼저 ErrStoreCorrupt로 끝나
// Save()에 도달하지 못한다. 그래서 Store를 직접 구성해 Save()만 실패시킨다.
func TestStore_SaveCleansUpTempFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	metaPath := filepath.Join(dir, "profiles.json")

	store, err := Load(metaPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := store.Add("work"); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// 목적지 자리에 디렉토리를 놓아 rename만 실패하게 만든다 — 임시 파일 쓰기는 이미 끝난
	// 뒤이므로 Save()의 정리 경로가 실제로 도는지 확인할 수 있다.
	if err := os.MkdirAll(metaPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", metaPath, err)
	}

	if err := store.Save(); err == nil {
		t.Fatalf("Save() error = nil, want failure because destination is occupied by a directory")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("leftover temp file after failed Save(): %s", e.Name())
		}
	}
}
