package cli_test

import (
	"slices"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// 등록된 이름으로 실행하면 조회로 확정된 경로에 "auth login" 뒤로 "--" 뒤 값이 순서·개수
// 그대로 붙은 인자가 넘어가고, 자식 환경은 use와 같은 규칙으로 채워진다.
func TestLogin_InjectsProfileConfigDirAndForwardsAuthLoginArgs(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}
	wantDir := listedDir(t, layout, "work")

	launcher := &recordingLauncher{path: fakeExecPath}
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	stdout, stderr, err := runUseCLI(t, layout, launcher, baseEnv, "login", "work", "--", "--console")
	if err != nil {
		t.Fatalf("login error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	// 성공 경로에서 ccswitch 자신은 한 줄도 내지 않는다.
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}

	if !slices.Equal(launcher.lookups, []string{"claude"}) {
		t.Errorf("lookups = %q, want [claude]", launcher.lookups)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("got %d launches, want 1", len(launcher.specs))
	}
	spec := launcher.specs[0]

	if spec.Path != fakeExecPath {
		t.Errorf("Spec.Path = %q, want %q", spec.Path, fakeExecPath)
	}
	if want := []string{"auth", "login", "--console"}; !slices.Equal(spec.Args, want) {
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
}

// default를 이름으로 주면 두 설정 디렉토리 변수 모두 자식 환경에 들어가지 않고, 넘어가는
// 인자는 "auth login"뿐이다.
func TestLogin_DefaultTargetLeavesNoConfigDirEnv(t *testing.T) {
	launcher := &recordingLauncher{path: fakeExecPath}
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	stdout, stderr, err := runUseCLI(t, newTestLayout(t), launcher, baseEnv, "login", "default")
	if err != nil {
		t.Fatalf("login error = %v, stderr = %q", err, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("got %d launches, want 1", len(launcher.specs))
	}
	spec := launcher.specs[0]

	for _, envName := range []string{"CLAUDE_CONFIG_DIR", "CLAUDE_SECURESTORAGE_CONFIG_DIR"} {
		if got := envValues(spec.Env, envName); len(got) != 0 {
			t.Errorf("%s values = %q, want none", envName, got)
		}
	}
	if want := []string{"auth", "login"}; !slices.Equal(spec.Args, want) {
		t.Errorf("Spec.Args = %q, want %q", spec.Args, want)
	}
}

// 위치 인자가 0개이거나 2개 이상이면 코드 2로 거부되고 실행 경계에 닿지 않는다 — login은
// use와 달리 이름 생략을 허용하지 않는다.
func TestLogin_RequiresExactlyOnePositionalArgument(t *testing.T) {
	cases := map[string][]string{
		"name omitted":               {"login"},
		"more than one profile name": {"login", "work", "personal"},
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
			if len(launcher.lookups) != 0 || len(launcher.specs) != 0 {
				t.Errorf("launcher was called: lookups=%q specs=%v", launcher.lookups, launcher.specs)
			}
		})
	}
}
