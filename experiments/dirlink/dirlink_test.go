// Package dirlink는 M3 착수 전 실험이다: 프로필 디렉토리 안의 디렉토리 링크(Windows junction /
// Unix symlink)를 os.RemoveAll이 따라 들어가 공유 원본을 지우는지 확인한다. main에 병합하지 않는다.
package dirlink_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func makeDirLink(t *testing.T, target, link string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/c", "mklink", "/J", link, target).CombinedOutput()
		if err != nil {
			t.Fatalf("mklink /J: %v: %s", err, out)
		}
		return
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func setup(t *testing.T) (shared, keep string) {
	t.Helper()
	root := t.TempDir()
	shared = filepath.Join(root, "shared", "commands")
	if err := os.MkdirAll(filepath.Join(shared, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	keep = filepath.Join(shared, "nested", "keep.md")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	return shared, keep
}

func logModes(t *testing.T, link string) {
	t.Helper()
	l, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	s, err := os.Stat(link)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %s: Lstat mode=%v IsDir=%v Symlink=%v Irregular=%v | Stat mode=%v",
		runtime.GOOS, runtime.Version(), l.Mode(), l.IsDir(),
		l.Mode()&os.ModeSymlink != 0, l.Mode()&os.ModeIrregular != 0, s.Mode())
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("shared original lost: %s: %v", path, err)
	}
}

func mustBeGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("%s still exists (err=%v)", path, err)
	}
}

// 프로필 디렉토리 안에 공유 디렉토리 링크가 있을 때 프로필 전체를 지운다 — 지금 rm이 하는 일.
func TestRemoveAll_ProfileContainingDirLink(t *testing.T) {
	shared, keep := setup(t)
	profile := filepath.Join(t.TempDir(), ".claude-work")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(profile, "commands")
	makeDirLink(t, shared, link)
	logModes(t, link)
	if _, err := os.Stat(filepath.Join(link, "nested", "keep.md")); err != nil {
		t.Fatalf("link does not resolve: %v", err)
	}

	if err := os.RemoveAll(profile); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	mustBeGone(t, profile)
	mustExist(t, keep)
}

// 링크 자체를 RemoveAll로 지운다 — sync 등에서 링크를 다시 만들 때의 경로.
func TestRemoveAll_LinkItself(t *testing.T) {
	shared, keep := setup(t)
	link := filepath.Join(t.TempDir(), "commands")
	makeDirLink(t, shared, link)

	if err := os.RemoveAll(link); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	mustBeGone(t, link)
	mustExist(t, keep)
}

// 링크를 os.Remove로 지운다 — 가장 좁은 수단이 링크만 지우는지.
func TestRemove_LinkItself(t *testing.T) {
	shared, keep := setup(t)
	link := filepath.Join(t.TempDir(), "commands")
	makeDirLink(t, shared, link)

	if err := os.Remove(link); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	mustBeGone(t, link)
	mustExist(t, keep)
}

// 링크 대상(공유 원본)이 이미 사라진 깨진 링크가 있어도 프로필 삭제가 실패하지 않는지.
func TestRemoveAll_ProfileContainingDanglingLink(t *testing.T) {
	shared, _ := setup(t)
	profile := filepath.Join(t.TempDir(), ".claude-work")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	makeDirLink(t, shared, filepath.Join(profile, "commands"))
	if err := os.RemoveAll(filepath.Dir(shared)); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(profile); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}
	mustBeGone(t, profile)
}
