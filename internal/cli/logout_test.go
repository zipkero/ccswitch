package cli_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
)

// 등록된 이름으로 실행하면 캡처 실행에 "auth logout"만 넘어가고(전달값 없음), 자식 환경에는
// 그 프로필 디렉토리가 CLAUDE_CONFIG_DIR로 한 번만 들어가며 CLAUDE_SECURESTORAGE_CONFIG_DIR은
// 들어가지 않는다. 성공하면 stdout은 비고 stderr 한 줄에 프로필 이름이 담기며 코드 0이다.
func TestLogout_CapturesAuthLogoutArgsAndInjectsProfileConfigDir(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}
	wantDir := listedDir(t, layout, "work")

	launcher := &recordingLauncher{
		path:          fakeExecPath,
		captureResult: launch.Captured{ExitCode: 0, Stdout: "Successfully logged out from your Anthropic account.\n"},
	}
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	stdout, stderr, err := runUseCLI(t, layout, launcher, baseEnv, "logout", "work")
	if err != nil {
		t.Fatalf("logout error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, `"work"`) {
		t.Errorf("stderr = %q, want it to mention the profile name", stderr)
	}

	if len(launcher.captures) != 1 {
		t.Fatalf("got %d captures, want 1", len(launcher.captures))
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
	if got := envValues(spec.Env, "PATH"); !slices.Equal(got, []string{"/usr/bin"}) {
		t.Errorf("PATH values = %q, want unrelated variables untouched", got)
	}
	// logout은 Run이 아니라 Capture만 써야 한다 — 콘솔을 넘기면 성공 줄이 어느 프로필인지
	// 말하지 않는다는 것이 이 명령을 캡처로 만든 이유다(D3).
	if len(launcher.specs) != 0 {
		t.Errorf("Run was called: specs = %v", launcher.specs)
	}
}

// default를 이름으로 주면 두 설정 디렉토리 변수 모두 자식 환경에 들어가지 않는다.
func TestLogout_DefaultTargetLeavesNoConfigDirEnvAndReportsDefaultDir(t *testing.T) {
	layout := newTestLayout(t)
	launcher := &recordingLauncher{
		path:          fakeExecPath,
		captureResult: launch.Captured{ExitCode: 0},
	}
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	stdout, stderr, err := runUseCLI(t, layout, launcher, baseEnv, "logout", "default")
	if err != nil {
		t.Fatalf("logout error = %v, stderr = %q", err, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, `"default"`) {
		t.Errorf("stderr = %q, want it to mention the profile name", stderr)
	}
	if len(launcher.captures) != 1 {
		t.Fatalf("got %d captures, want 1", len(launcher.captures))
	}
	spec := launcher.captures[0]

	for _, envName := range []string{"CLAUDE_CONFIG_DIR", "CLAUDE_SECURESTORAGE_CONFIG_DIR"} {
		if got := envValues(spec.Env, envName); len(got) != 0 {
			t.Errorf("%s values = %q, want none", envName, got)
		}
	}
	if want := []string{"auth", "logout"}; !slices.Equal(spec.Args, want) {
		t.Errorf("Spec.Args = %q, want %q", spec.Args, want)
	}
}

// 캡처 결과 종료 코드가 0이 아니거나 자식을 띄우지 못하면 코드 8이고, stderr에 캡처한 출력이
// 나른 이유와 사용자가 취할 조치가 들어간다.
func TestLogout_CaptureFailureExitsWithCode8(t *testing.T) {
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

			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "logout", "work")
			if err == nil {
				t.Fatalf("logout error = nil, want a failure")
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
			// 상태를 바꾸는 명령을 조치로 권하지 않는다.
			if strings.Contains(stderr, "ccswitch login") {
				t.Errorf("stderr = %q, want it not to suggest a state-changing command", stderr)
			}
			if len(launcher.captures) != 1 {
				t.Errorf("got %d captures, want 1", len(launcher.captures))
			}
		})
	}
}

// 위치 인자가 0개거나 2개 이상이면 코드 2로 거부되고 실행 경계에 닿지 않는다.
func TestLogout_RequiresExactlyOnePositionalArgument(t *testing.T) {
	cases := map[string][]string{
		"name omitted":               {"logout"},
		"more than one profile name": {"logout", "work", "personal"},
	}

	for name, argv := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			launcher := &recordingLauncher{path: fakeExecPath}

			_, stderr, err := runUseCLI(t, layout, launcher, nil, argv...)
			if err == nil {
				t.Fatalf("%v error = nil, want usage error", argv)
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
			if stderr == "" {
				t.Errorf("stderr is empty, want a message")
			}
			if len(launcher.lookups) != 0 || len(launcher.captures) != 0 {
				t.Errorf("launcher was called: lookups=%q captures=%v", launcher.lookups, launcher.captures)
			}
		})
	}
}

// logout은 use·login과 달리 "--" 뒤 전달을 받지 않는다 — 무엇을 주든 코드 2이고 프로세스가
// 뜨지 않는다.
func TestLogout_RejectsAnythingAfterDashDash(t *testing.T) {
	cases := map[string][]string{
		"flag after dash dash":       {"logout", "work", "--", "--console"},
		"name given after dash dash": {"logout", "--", "work"},
	}

	for name, argv := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			launcher := &recordingLauncher{path: fakeExecPath}

			_, stderr, err := runUseCLI(t, layout, launcher, nil, argv...)

			if err == nil {
				t.Fatalf("%v error = nil, want usage error", argv)
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
			if stderr == "" {
				t.Errorf("stderr is empty, want a message")
			}
			if len(launcher.lookups) != 0 || len(launcher.captures) != 0 {
				t.Errorf("launcher was called: lookups=%q captures=%v", launcher.lookups, launcher.captures)
			}
		})
	}
}

// "--"가 있어도 그 뒤에 아무것도 없으면 전달할 것이 없으므로, 앞의 유일한 위치 인자가 그대로
// 이름으로 받아들여진다 — 거부되는 것은 "--" 뒤에 무언가가 있을 때뿐이다.
func TestLogout_AcceptsDashDashWithNothingAfterIt(t *testing.T) {
	layout := newTestLayout(t)
	launcher := &recordingLauncher{
		path:          fakeExecPath,
		captureResult: launch.Captured{ExitCode: 0},
	}

	_, stderr, err := runUseCLI(t, layout, launcher, nil, "logout", "default", "--")
	if err != nil {
		t.Fatalf(`logout "default" "--" error = %v, stderr = %q`, err, stderr)
	}
	if len(launcher.captures) != 1 {
		t.Errorf("got %d captures, want 1", len(launcher.captures))
	}
}
