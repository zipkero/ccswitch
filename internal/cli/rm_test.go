package cli_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// runRmCLI는 Stdin·Interactive까지 값으로 채운 Deps로 새 커맨드 트리를 구성해 실행한다.
// runCommand(list_test.go)는 이 두 필드를 다루지 않으므로 rm 전용으로 따로 둔다.
func runRmCLI(t *testing.T, layout profile.Layout, stdin io.Reader, interactive bool, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	root := cli.NewRootCommand(cli.Deps{
		Layout:      layout,
		Stdin:       stdin,
		Stdout:      &outBuf,
		Stderr:      &errBuf,
		Interactive: interactive,
	})
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// (1) --yes로 삭제하면 디렉토리·등록이 모두 사라지고 list에 나타나지 않는다.
func TestRm_WithYesFlag_RemovesDirectoryAndRegistration(t *testing.T) {
	layout := newTestLayout(t)
	if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v", err)
	}

	_, _, err := runRmCLI(t, layout, nil, false, "rm", "work", "--yes")
	if err != nil {
		t.Fatalf("rm error = %v", err)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}

	if _, statErr := os.Stat(layout.ProfileDir("work")); !os.IsNotExist(statErr) {
		t.Errorf("ProfileDir() stat error = %v, want it to not exist", statErr)
	}

	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	for _, r := range dataRows(t, stdout) {
		if r[0] == "work" {
			t.Errorf("list still shows removed profile: %v", r)
		}
	}
}

// (2) 대화형으로 "n"이나 빈 입력을 주면 아무것도 지워지지 않고 종료 코드 0, stderr에 취소
// 표시가 남는다.
func TestRm_InteractiveDecline_KeepsStateAndExitsZero(t *testing.T) {
	inputs := map[string]string{
		"explicit no": "n\n",
		"empty input": "\n",
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
				t.Fatalf("add error = %v", err)
			}

			dirBefore, err := os.ReadDir(layout.ProfileDir("work"))
			if err != nil {
				t.Fatalf("ReadDir() error = %v", err)
			}
			registryBefore, err := os.ReadFile(layout.MetadataPath())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			_, stderr, err := runRmCLI(t, layout, strings.NewReader(input), true, "rm", "work")
			if err != nil {
				t.Fatalf("rm error = %v", err)
			}
			if code := cli.ExitCode(err); code != 0 {
				t.Errorf("ExitCode() = %d, want 0", code)
			}
			if !strings.Contains(stderr, "Cancelled") {
				t.Errorf("stderr = %q, want it to mention cancellation", stderr)
			}

			dirAfter, err := os.ReadDir(layout.ProfileDir("work"))
			if err != nil {
				t.Fatalf("ReadDir() error = %v", err)
			}
			if len(dirAfter) != len(dirBefore) {
				t.Errorf("directory entries changed: before=%v after=%v", dirBefore, dirAfter)
			}
			registryAfter, err := os.ReadFile(layout.MetadataPath())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(registryBefore) != string(registryAfter) {
				t.Errorf("registry file changed:\nbefore=%q\nafter=%q", registryBefore, registryAfter)
			}
		})
	}
}

// (3) 대화형으로 "y"를 주면 삭제가 성공한다.
func TestRm_InteractiveApprove_RemovesState(t *testing.T) {
	layout := newTestLayout(t)
	if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v", err)
	}

	_, _, err := runRmCLI(t, layout, strings.NewReader("y\n"), true, "rm", "work")
	if err != nil {
		t.Fatalf("rm error = %v", err)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	if _, statErr := os.Stat(layout.ProfileDir("work")); !os.IsNotExist(statErr) {
		t.Errorf("ProfileDir() stat error = %v, want it to not exist", statErr)
	}
}

// (4) 비대화형에서 --yes가 없으면 아무것도 지우지 않고 종료 코드 2, stderr에 생략 플래그
// 안내가 남는다.
func TestRm_NonInteractiveWithoutFlag_FailsAndKeepsState(t *testing.T) {
	layout := newTestLayout(t)
	if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v", err)
	}

	dirBefore, err := os.ReadDir(layout.ProfileDir("work"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	registryBefore, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	_, stderr, err := runRmCLI(t, layout, nil, false, "rm", "work")
	if err == nil {
		t.Fatalf("rm error = nil, want failure")
	}
	if code := cli.ExitCode(err); code != 2 {
		t.Errorf("ExitCode() = %d, want 2", code)
	}
	if !strings.Contains(stderr, "--yes") {
		t.Errorf("stderr = %q, want it to mention --yes", stderr)
	}

	dirAfter, err := os.ReadDir(layout.ProfileDir("work"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(dirAfter) != len(dirBefore) {
		t.Errorf("directory entries changed: before=%v after=%v", dirBefore, dirAfter)
	}
	registryAfter, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(registryBefore) != string(registryAfter) {
		t.Errorf("registry file changed:\nbefore=%q\nafter=%q", registryBefore, registryAfter)
	}
}

// (5) 등록 파일이 손상되면 아무것도 지우지 않고 그 파일 경로를 알리며 실패한다.
func TestRm_FailsOnCorruptedRegistryFileWithoutDeletingAnything(t *testing.T) {
	layout := newTestLayout(t)
	if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v", err)
	}

	corrupt := []byte("not valid json")
	if err := os.WriteFile(layout.MetadataPath(), corrupt, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, stderr, err := runRmCLI(t, layout, nil, false, "rm", "work", "--yes")
	if err == nil {
		t.Fatalf("rm error = nil, want failure")
	}
	if code := cli.ExitCode(err); code != 1 {
		t.Errorf("ExitCode() = %d, want 1", code)
	}
	if !strings.Contains(stderr, layout.MetadataPath()) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, layout.MetadataPath())
	}

	if _, statErr := os.Stat(layout.ProfileDir("work")); statErr != nil {
		t.Errorf("ProfileDir() stat error = %v, want the directory to still exist", statErr)
	}
	after, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(after) != string(corrupt) {
		t.Errorf("registry file changed:\nbefore=%q\nafter=%q", corrupt, after)
	}
}

// (6) 대상 프로필 디렉토리 안에 파일과 하위 디렉토리가 있어도 삭제가 완료되고, 홈 아래의
// 다른 프로필 디렉토리와 기본 설정 디렉토리는 그대로 남는다 — 삭제 범위가 대상 하나에만
// 미치는지가 이 Task에서 가장 중요한 확인이다.
func TestRm_OnlyRemovesTargetProfile_LeavesOthersAndDefaultUntouched(t *testing.T) {
	layout := newTestLayout(t)

	if _, _, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add work error = %v", err)
	}
	workDir := layout.ProfileDir("work")
	if err := os.MkdirAll(filepath.Join(workDir, "sub"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "sub", "nested.txt"), []byte("nested"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, _, err := runCommand(t, layout, "add", "personal"); err != nil {
		t.Fatalf("add personal error = %v", err)
	}
	personalKeep := filepath.Join(layout.ProfileDir("personal"), "keep.txt")
	if err := os.WriteFile(personalKeep, []byte("keep"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	defaultDir := layout.DefaultDir()
	defaultKeep := filepath.Join(defaultDir, "settings.json")
	if err := os.MkdirAll(defaultDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(defaultKeep, []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, stderr, err := runRmCLI(t, layout, nil, false, "rm", "work", "--yes"); err != nil {
		t.Fatalf("rm error = %v, stderr = %q", err, stderr)
	}

	if _, statErr := os.Stat(workDir); !os.IsNotExist(statErr) {
		t.Errorf("workDir stat error = %v, want it to not exist", statErr)
	}

	if content, err := os.ReadFile(personalKeep); err != nil || string(content) != "keep" {
		t.Errorf("personal profile touched: content=%q err=%v", content, err)
	}
	if content, err := os.ReadFile(defaultKeep); err != nil || string(content) != "{}" {
		t.Errorf("default dir touched: content=%q err=%v", content, err)
	}

	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	names := map[string]bool{}
	for _, r := range dataRows(t, stdout) {
		names[r[0]] = true
	}
	if names["work"] {
		t.Errorf("list still shows removed profile: %v", names)
	}
	if !names["personal"] {
		t.Errorf("list is missing untouched profile %q: %v", "personal", names)
	}
	if !names["default"] {
		t.Errorf("list is missing the default entry: %v", names)
	}
}
