package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// newTestLayout은 실제 홈·설정 위치를 건드리지 않도록 임시 디렉토리 두 개로 Layout을
// 구성한다. 커맨드 트리도 매번 새로 만들어 전역 상태가 테스트 사이에 남지 않는다.
func newTestLayout(t *testing.T) profile.Layout {
	t.Helper()
	return profile.Layout{
		Home:       t.TempDir(),
		ConfigRoot: t.TempDir(),
	}
}

// runList는 새 커맨드 트리를 구성해 `list`를 실행하고 stdout·stderr를 값으로 돌려준다.
func runList(t *testing.T, layout profile.Layout) (stdout, stderr string, err error) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	root := cli.NewRootCommand(cli.Deps{Layout: layout, Stdout: &outBuf, Stderr: &errBuf})
	root.SetArgs([]string{"list"})
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// dataRows는 표 출력에서 헤더를 제외한 데이터 행을 필드 단위로 쪼갠다.
func dataRows(t *testing.T, stdout string) [][]string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("stdout has no lines")
	}
	if fields := strings.Fields(lines[0]); len(fields) != 3 || fields[0] != "NAME" {
		t.Fatalf("unexpected header line: %q", lines[0])
	}
	rows := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			t.Fatalf("unexpected data line %q: got %d fields", line, len(fields))
		}
		rows = append(rows, fields)
	}
	return rows
}

func TestList_NoRegistrations(t *testing.T) {
	layout := newTestLayout(t)

	stdout, _, err := runList(t, layout)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Fatalf("ExitCode() = %d, want 0", code)
	}

	rows := dataRows(t, stdout)
	if len(rows) != 1 {
		t.Fatalf("got %d data rows, want 1 (default only): %v", len(rows), rows)
	}
	if rows[0][0] != profile.DefaultName {
		t.Errorf("row name = %q, want %q", rows[0][0], profile.DefaultName)
	}
	if rows[0][1] != layout.DefaultDir() {
		t.Errorf("row dir = %q, want %q", rows[0][1], layout.DefaultDir())
	}
}

func TestList_RegisteredProfilesSortedWithDefaultFirst(t *testing.T) {
	layout := newTestLayout(t)

	store, err := profile.Load(layout.MetadataPath())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	// "apple"은 사전순으로 "default"보다 앞서고 "zulu"는 뒤에 온다 — default가
	// 알파벳 순서가 아니라 항상 맨 위에 고정됨을 이 배치로 확인한다.
	for _, name := range []string{"zulu", "apple"} {
		if err := store.Add(name); err != nil {
			t.Fatalf("Add(%q) error = %v", name, err)
		}
		if err := os.MkdirAll(layout.ProfileDir(name), 0o755); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", name, err)
		}
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	stdout, _, err := runList(t, layout)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rows := dataRows(t, stdout)
	wantNames := []string{profile.DefaultName, "apple", "zulu"}
	if len(rows) != len(wantNames) {
		t.Fatalf("got %d data rows, want %d: %v", len(rows), len(wantNames), rows)
	}
	for i, want := range wantNames {
		if rows[i][0] != want {
			t.Errorf("row %d name = %q, want %q", i, rows[i][0], want)
		}
	}
	// 등록된 두 이름은 디렉토리를 만들어 두었으므로 ok여야 한다.
	if rows[1][2] != "ok" || rows[2][2] != "ok" {
		t.Errorf("registered rows status = %q, %q, want both ok", rows[1][2], rows[2][2])
	}
}

func TestList_DistinguishesMissingAndUnusable(t *testing.T) {
	layout := newTestLayout(t)

	store, err := profile.Load(layout.MetadataPath())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	for _, name := range []string{"beta", "gone", "occupied"} {
		if err := store.Add(name); err != nil {
			t.Fatalf("Add(%q) error = %v", name, err)
		}
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// beta: 정상 디렉토리 -> ok
	if err := os.MkdirAll(layout.ProfileDir("beta"), 0o755); err != nil {
		t.Fatalf("MkdirAll(beta) error = %v", err)
	}
	// gone: 등록만 있고 디렉토리는 밖에서 지워진 것처럼 만들지 않는다 -> missing
	// occupied: 디렉토리 자리에 파일이 놓임 -> unusable
	if err := os.WriteFile(layout.ProfileDir("occupied"), []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile(occupied) error = %v", err)
	}

	stdout, _, err := runList(t, layout)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	rows := dataRows(t, stdout)
	status := make(map[string]string, len(rows))
	for _, r := range rows {
		status[r[0]] = r[2]
	}
	want := map[string]string{"beta": "ok", "gone": "missing", "occupied": "unusable"}
	for name, wantStatus := range want {
		if got := status[name]; got != wantStatus {
			t.Errorf("status[%q] = %q, want %q", name, got, wantStatus)
		}
	}
}

func TestList_CorruptedStoreFailsWithoutPrintingTable(t *testing.T) {
	// 등록 파일이 손상된 것으로 간주되는 두 경로 — 파싱 자체가 안 되는 경우와, 파싱은
	// 되지만 이 코드가 모르는 version인 경우 — 모두 빈 목록으로 넘어가지 않고 실패해야 한다.
	cases := []struct {
		name    string
		content string
	}{
		{name: "malformed json", content: "not valid json"},
		{name: "unknown version", content: `{"version": 2, "profiles": []}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			layout := newTestLayout(t)

			metaPath := layout.MetadataPath()
			if err := os.MkdirAll(filepath.Dir(metaPath), 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			if err := os.WriteFile(metaPath, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			stdout, stderr, err := runList(t, layout)
			if err == nil {
				t.Fatalf("Execute() error = nil, want failure for corrupted store")
			}
			if code := cli.ExitCode(err); code != 1 {
				t.Errorf("ExitCode() = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty (no table on failure)", stdout)
			}
			if !strings.Contains(stderr, metaPath) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, metaPath)
			}
		})
	}
}
