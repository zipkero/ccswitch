package cli_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
)

// 승인 뒤 캡처 실행이 정확히 한 번 일어나고, 그 인자가 "auth logout"이며 자식 환경의
// CLAUDE_CONFIG_DIR이 지워질 프로필 디렉토리 하나다.
func TestRm_DelegatesLogoutArgsAndConfigDirBeforeDeleting(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}
	wantDir := listedDir(t, layout, "work")

	launcher := &recordingLauncher{
		path:          fakeExecPath,
		captureResult: launch.Captured{ExitCode: 0},
	}
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	root := cli.NewRootCommand(cli.Deps{
		Layout:   layout,
		Stdout:   new(strings.Builder),
		Stderr:   new(strings.Builder),
		Platform: launch.NewPlatform(),
		BaseEnv:  baseEnv,
		Launcher: launcher,
	})
	root.SetArgs([]string{"rm", "work", "--yes"})
	if err := root.Execute(); err != nil {
		t.Fatalf("rm error = %v", err)
	}

	if len(launcher.captures) != 1 {
		t.Fatalf("got %d captures, want exactly 1", len(launcher.captures))
	}
	spec := launcher.captures[0]

	if spec.Path != fakeExecPath {
		t.Errorf("Spec.Path = %q, want %q", spec.Path, fakeExecPath)
	}
	if want := []string{"auth", "logout"}; !slices.Equal(spec.Args, want) {
		t.Errorf("Spec.Args = %q, want %q", spec.Args, want)
	}
	if got := envValues(spec.Env, "CLAUDE_CONFIG_DIR"); !slices.Equal(got, []string{wantDir}) {
		t.Errorf("CLAUDE_CONFIG_DIR values = %q, want exactly one %q", got, wantDir)
	}
	if got := envValues(spec.Env, "CLAUDE_SECURESTORAGE_CONFIG_DIR"); len(got) != 0 {
		t.Errorf("CLAUDE_SECURESTORAGE_CONFIG_DIR values = %q, want none", got)
	}
	// rm은 Run이 아니라 Capture만 써야 한다 — 삭제 전 정리는 화면에 콘솔을 넘길 이유가 없다.
	if len(launcher.specs) != 0 {
		t.Errorf("Run was called: specs = %v", launcher.specs)
	}
}

// 위임이 0이 아닌 코드로 끝나거나 자식을 띄우지 못하면 코드 8이고, 디렉토리·등록·list 출력이
// 실행 전과 같으며 stderr에 캡처한 이유와 조치가 들어간다.
func TestRm_DelegationFailure_KeepsStateAndExitsWithCode8(t *testing.T) {
	cases := map[string]*recordingLauncher{
		"non-zero exit code": {
			path: fakeExecPath,
			captureResult: launch.Captured{
				ExitCode: 1,
				Stderr:   "some claude auth logout failure",
			},
		},
		"could not launch": {
			path:       fakeExecPath,
			captureErr: errors.New("fork/exec: permission denied"),
		},
	}

	for name, launcher := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
				t.Fatalf("add error = %v, stderr = %q", err, stderr)
			}

			wantStore := storeSnapshot(t, layout)
			wantList := listSnapshot(t, layout)
			dirBefore, err := os.ReadDir(layout.ProfileDir("work"))
			if err != nil {
				t.Fatalf("ReadDir() error = %v", err)
			}

			stdout, stderr, err := runRmCLIWithLauncher(t, layout, nil, false, launcher, "rm", "work", "--yes")
			if err == nil {
				t.Fatalf("rm error = nil, want a failure")
			}
			if code := cli.ExitCode(err); code != 8 {
				t.Errorf("ExitCode() = %d, want 8", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr == "" {
				t.Errorf("stderr is empty, want a reason and an action")
			}
			if !strings.Contains(stderr, fakeExecPath) {
				t.Errorf("stderr = %q, want it to mention the installation to check", stderr)
			}
			// 실패 원인은 위임한 명령이 답을 주지 못한 것이지 프로필의 로그인 여부가 아니므로,
			// 상태를 바꾸는 명령을 조치로 권하지 않는다(logout과 같은 기준).
			if strings.Contains(stderr, "ccswitch login") {
				t.Errorf("stderr = %q, want it not to suggest a state-changing command", stderr)
			}
			if len(launcher.captures) != 1 {
				t.Errorf("got %d captures, want exactly 1", len(launcher.captures))
			}

			if _, statErr := os.Stat(layout.ProfileDir("work")); statErr != nil {
				t.Errorf("ProfileDir() stat error = %v, want the directory to still exist", statErr)
			}
			dirAfter, err := os.ReadDir(layout.ProfileDir("work"))
			if err != nil {
				t.Fatalf("ReadDir() error = %v", err)
			}
			if len(dirAfter) != len(dirBefore) {
				t.Errorf("directory entries changed: before=%v after=%v", dirBefore, dirAfter)
			}
			if got := storeSnapshot(t, layout); got != wantStore {
				t.Errorf("store file changed:\nbefore = %q\nafter  = %q", wantStore, got)
			}
			if got := listSnapshot(t, layout); got != wantList {
				t.Errorf("list output changed:\nbefore = %q\nafter  = %q", wantList, got)
			}
		})
	}
}

// PATH에서 실행 파일을 찾지 못하면 코드 7이고, 확인 프롬프트 문구는 화면에 나오지 않으며
// 디렉토리·등록·list 출력이 실행 전과 같다 — PATH 조회가 프롬프트보다
// 앞이므로(profile-auth D9) 실행할 수 없는 정리를 두고 승인을 묻지 않는다.
func TestRm_ExecutableNotFound_RejectsBeforePromptingAndKeepsState(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	wantStore := storeSnapshot(t, layout)
	wantList := listSnapshot(t, layout)

	launcher := &notFoundLauncher{}
	// --yes를 주지 않고 비대화형으로 돌린다 — 조회 실패가 확인 프롬프트보다 먼저 걸리는지가
	// 이 케이스의 핵심이므로, 프롬프트 자체에 닿을 다른 경로(대화형 승인 등)를 배제한다.
	_, stderr, err := runRmCLIWithLauncher(t, layout, nil, false, launcher, "rm", "work")
	if err == nil {
		t.Fatalf("rm error = nil, want a rejection")
	}
	if code := cli.ExitCode(err); code != 7 {
		t.Errorf("ExitCode() = %d, want 7", code)
	}
	if strings.Contains(stderr, "Remove profile") {
		t.Errorf("stderr = %q, want no confirmation prompt", stderr)
	}
	if len(launcher.captures) != 0 {
		t.Errorf("delegation ran despite the lookup failure: captures = %v", launcher.captures)
	}

	if got := storeSnapshot(t, layout); got != wantStore {
		t.Errorf("store file changed:\nbefore = %q\nafter  = %q", wantStore, got)
	}
	if got := listSnapshot(t, layout); got != wantList {
		t.Errorf("list output changed:\nbefore = %q\nafter  = %q", wantList, got)
	}
}

// 등록만 있고 디렉토리가 없는 프로필에도 정리 위임이 한 번 일어나고 삭제가 끝난다 — claude
// auth logout은 없는 설정 디렉토리에서도 성공으로 끝나고(profile-auth §근거), macOS에서
// 자격증명은 디렉토리 밖 Keychain에 있으므로 디렉토리 상태로 위임을 걸러내면 안 된다
// (profile-auth D10).
func TestRm_RegisteredWithMissingDirectory_DelegatesOnceThenRemoves(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}
	if err := os.RemoveAll(layout.ProfileDir("work")); err != nil {
		t.Fatalf("RemoveAll() error = %v", err)
	}

	launcher := &recordingLauncher{
		path:          fakeExecPath,
		captureResult: launch.Captured{ExitCode: 0},
	}
	_, stderr, err := runRmCLIWithLauncher(t, layout, nil, false, launcher, "rm", "work", "--yes")
	if err != nil {
		t.Fatalf("rm error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	if len(launcher.captures) != 1 {
		t.Errorf("got %d captures, want exactly 1", len(launcher.captures))
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
