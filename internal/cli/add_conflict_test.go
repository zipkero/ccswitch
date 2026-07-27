package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// 이미 등록된 이름으로 다시 add하면 기존 디렉토리·등록 내용이 그대로여야 한다(SPEC §5.4).
func TestAdd_RejectsDuplicateNameWithoutChangingExistingState(t *testing.T) {
	layout := newTestLayout(t)

	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("first add error = %v, stderr = %q", err, stderr)
	}

	dir := layout.ProfileDir("work")
	entriesBefore, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	registryBefore, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	_, stderr, err := runCommand(t, layout, "add", "work")
	if err == nil {
		t.Fatalf("second add error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 4 {
		t.Errorf("ExitCode() = %d, want 4", code)
	}
	// 목적 필드가 이미 등록된 이름과 점유된 경로 두 경우 모두 조치 안내를 요구한다.
	if !strings.Contains(stderr, "different profile name") {
		t.Errorf("stderr = %q, want it to suggest a different profile name", stderr)
	}

	entriesAfter, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir(%q) error = %v", dir, err)
	}
	if len(entriesAfter) != len(entriesBefore) {
		t.Errorf("directory entries changed: before=%v after=%v", entriesBefore, entriesAfter)
	}

	registryAfter, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(registryBefore) != string(registryAfter) {
		t.Errorf("registry file changed:\nbefore=%q\nafter=%q", registryBefore, registryAfter)
	}
}

// 대상 자리에 파일이 하나 든 디렉토리가 이미 있으면 그 내용을 건드리지 않고 실패해야 한다
// (SPEC §5.5).
func TestAdd_RejectsWhenTargetDirHasContents(t *testing.T) {
	layout := newTestLayout(t)
	dir := layout.ProfileDir("work")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}
	keepPath := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(keepPath, []byte("keep me"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, err := runCommand(t, layout, "add", "work")
	if err == nil {
		t.Fatalf("add error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 5 {
		t.Errorf("ExitCode() = %d, want 5", code)
	}
	if !strings.Contains(stderr, dir) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, dir)
	}

	content, err := os.ReadFile(keepPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", keepPath, err)
	}
	if string(content) != "keep me" {
		t.Errorf("file content changed: %q", content)
	}
	if _, statErr := os.Stat(layout.MetadataPath()); !os.IsNotExist(statErr) {
		t.Errorf("MetadataPath() stat error = %v, want it to still not exist", statErr)
	}
}

// 대상 자리에 디렉토리가 아니라 같은 이름의 파일이 있어도 그 파일을 건드리지 않고 실패해야
// 한다(SPEC §5.5).
func TestAdd_RejectsWhenTargetIsAFile(t *testing.T) {
	layout := newTestLayout(t)
	dir := layout.ProfileDir("work")
	if err := os.WriteFile(dir, []byte("i am a file"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", dir, err)
	}

	_, stderr, err := runCommand(t, layout, "add", "work")
	if err == nil {
		t.Fatalf("add error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 5 {
		t.Errorf("ExitCode() = %d, want 5", code)
	}
	if !strings.Contains(stderr, dir) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, dir)
	}

	content, err := os.ReadFile(dir)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", dir, err)
	}
	if string(content) != "i am a file" {
		t.Errorf("file content changed: %q", content)
	}
}

// 등록 파일이 손상된 상태에서는 빈 목록으로 진행하지 않고 그 파일 경로를 알리며 멈춰야
// 한다(SPEC §5.11) — add에서도 list와 같은 규칙이 성립해야 한다.
func TestAdd_FailsOnCorruptedRegistryFileWithoutChangingItOrCreatingADirectory(t *testing.T) {
	layout := newTestLayout(t)
	metaPath := layout.MetadataPath()
	if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	original := []byte("not valid json")
	if err := os.WriteFile(metaPath, original, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, err := runCommand(t, layout, "add", "work")
	if err == nil {
		t.Fatalf("add error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 1 {
		t.Errorf("ExitCode() = %d, want 1", code)
	}
	if !strings.Contains(stderr, metaPath) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, metaPath)
	}

	after, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(after) != string(original) {
		t.Errorf("registry file changed:\nbefore=%q\nafter=%q", original, after)
	}

	if _, statErr := os.Stat(layout.ProfileDir("work")); !os.IsNotExist(statErr) {
		t.Errorf("ProfileDir() stat error = %v, want it to not exist", statErr)
	}
}
