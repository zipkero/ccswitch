package cli_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// runCommand는 임의의 서브커맨드 인자로 새 커맨드 트리를 구성해 실행하고 stdout·stderr를
// 값으로 돌려준다. newTestLayout·dataRows는 list_test.go에 있다.
func runCommand(t *testing.T, layout profile.Layout, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	root := cli.NewRootCommand(cli.Deps{Layout: layout, Stdout: &outBuf, Stderr: &errBuf})
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

func TestAdd_CreatesDirectoryAndAppearsInList(t *testing.T) {
	layout := newTestLayout(t)

	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf(`add "work" error = %v, stderr = %q`, err, stderr)
	}

	dir := layout.ProfileDir("work")
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat(%q) error = %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", dir)
	}

	// list를 새 커맨드 트리로 다시 실행해, add가 쓴 등록 파일이 그대로 다시 읽히는지
	// (왕복) 함께 확인한다.
	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	rows := dataRows(t, stdout)
	var found bool
	for _, r := range rows {
		if r[0] != "work" {
			continue
		}
		found = true
		if r[1] != dir {
			t.Errorf("row dir = %q, want %q", r[1], dir)
		}
		if r[2] != "ok" {
			t.Errorf("row status = %q, want ok", r[2])
		}
	}
	if !found {
		t.Fatalf(`list output missing "work" row: %v`, rows)
	}
}

func TestAdd_PassesWhenEmptyDirectoryAlreadyExists(t *testing.T) {
	layout := newTestLayout(t)
	dir := layout.ProfileDir("work")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}

	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf(`add "work" error = %v, stderr = %q`, err, stderr)
	}

	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	rows := dataRows(t, stdout)
	for _, r := range rows {
		if r[0] == "work" && r[2] != "ok" {
			t.Errorf("row status = %q, want ok", r[2])
		}
	}
}
