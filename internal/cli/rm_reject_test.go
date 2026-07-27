package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// (1) rm default는 프롬프트나 --yes 여부와 무관하게 항상 거부되고, 기본 설정 디렉토리와
// 안의 파일이 그대로 남아야 한다.
func TestRm_RejectsDefaultWithoutTouchingDefaultDir(t *testing.T) {
	cases := map[string][]string{
		"without --yes": {"rm", "default"},
		"with --yes":    {"rm", "default", "--yes"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			defaultDir := layout.DefaultDir()
			if err := os.MkdirAll(defaultDir, 0o755); err != nil {
				t.Fatalf("MkdirAll(%q) error = %v", defaultDir, err)
			}
			keepPath := filepath.Join(defaultDir, "settings.json")
			if err := os.WriteFile(keepPath, []byte("{}"), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			_, stderr, err := runRmCLI(t, layout, nil, false, args...)
			if err == nil {
				t.Fatalf("rm error = nil, want rejection")
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
			if strings.Contains(stderr, "Remove profile") {
				t.Errorf("stderr = %q, want no confirmation prompt", stderr)
			}

			content, err := os.ReadFile(keepPath)
			if err != nil {
				t.Fatalf("default dir was touched: %v", err)
			}
			if string(content) != "{}" {
				t.Errorf("default dir file changed: %q", content)
			}
		})
	}
}

// (2) 등록되지 않은 이름으로 rm하면 종료 코드 3, stderr에 찾을 수 없다는 내용이 나오고
// 홈 아래 어떤 디렉토리도 사라지지 않아야 한다.
func TestRm_RejectsUnregisteredNameWithoutDeletingAnything(t *testing.T) {
	layout := newTestLayout(t)
	if _, _, err := runCommand(t, layout, "add", "existing"); err != nil {
		t.Fatalf("add error = %v", err)
	}

	entriesBefore, err := os.ReadDir(layout.Home)
	if err != nil {
		t.Fatalf("ReadDir(Home) error = %v", err)
	}

	_, stderr, err := runRmCLI(t, layout, nil, false, "rm", "ghost", "--yes")
	if err == nil {
		t.Fatalf("rm error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 3 {
		t.Errorf("ExitCode() = %d, want 3", code)
	}
	if !strings.Contains(stderr, "not") {
		t.Errorf("stderr = %q, want it to say the profile could not be found", stderr)
	}
	if strings.Contains(stderr, "Remove profile") {
		t.Errorf("stderr = %q, want no confirmation prompt", stderr)
	}

	entriesAfter, err := os.ReadDir(layout.Home)
	if err != nil {
		t.Fatalf("ReadDir(Home) error = %v", err)
	}
	if len(entriesAfter) != len(entriesBefore) {
		t.Errorf("Home entries changed: before=%v after=%v", entriesBefore, entriesAfter)
	}
}

// (3) 규칙 위반 이름으로 rm하면 종료 코드 2이고 프롬프트가 출력되지 않는다.
func TestRm_RejectsInvalidNameWithoutPrompting(t *testing.T) {
	layout := newTestLayout(t)

	_, stderr, err := runRmCLI(t, layout, nil, false, "rm", "Bad Name", "--yes")
	if err == nil {
		t.Fatalf("rm error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 2 {
		t.Errorf("ExitCode() = %d, want 2", code)
	}
	if strings.Contains(stderr, "Remove profile") {
		t.Errorf("stderr = %q, want no confirmation prompt", stderr)
	}
}

// 회귀: 등록되지 않은 이름의 디렉토리가 홈에 존재해도(예: 이전 add가 디렉토리 생성 뒤 저장에
// 실패해 남긴 상태), 등록 확인이 삭제보다 먼저 끝나야 하므로 그 디렉토리가 지워지지 않아야
// 한다. 등록 확인보다 삭제가 먼저 실행되던 이전 순서에서는 이 케이스가 --yes로 조용히
// 디렉토리를 지워버렸다.
func TestRm_RegressionUnregisteredNameWithExistingDirectory_DoesNotDeleteIt(t *testing.T) {
	layout := newTestLayout(t)
	dir := layout.ProfileDir("ghost")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", dir, err)
	}
	markerPath := filepath.Join(dir, "marker.txt")
	if err := os.WriteFile(markerPath, []byte("keep"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, err := runRmCLI(t, layout, nil, false, "rm", "ghost", "--yes")
	if err == nil {
		t.Fatalf("rm error = nil, want rejection")
	}
	if code := cli.ExitCode(err); code != 3 {
		t.Errorf("ExitCode() = %d, want 3", code)
	}
	if !strings.Contains(stderr, "not") {
		t.Errorf("stderr = %q, want it to say the profile could not be found", stderr)
	}

	if _, statErr := os.Stat(dir); statErr != nil {
		t.Fatalf("ghost directory was deleted: %v", statErr)
	}
	content, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("marker file was deleted: %v", err)
	}
	if string(content) != "keep" {
		t.Errorf("marker file content changed: %q", content)
	}
}
