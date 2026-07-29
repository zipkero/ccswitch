package cli_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

// fakeExecPath는 조회가 확정한 것으로 가장할 실행 파일 경로다. 대역이 프로세스를 띄우지 않으므로
// 실제로 있는 파일일 필요가 없고, 그 값이 Spec.Path로 그대로 넘어가는지만 본다.
const fakeExecPath = "/fake/bin/claude"

// recordingLauncher는 launch.Launcher를 구현하며 넘겨받은 것만 기록한다. 프로세스를 띄우지
// 않으므로 "무슨 경로에 무슨 인자와 무슨 환경을 넘겼는가"를 프로세스 전역 PATH·환경변수를
// 건드리지 않고 확인할 수 있다(D1).
type recordingLauncher struct {
	path     string
	exitCode int

	lookups []string
	specs   []launch.Spec
}

func (l *recordingLauncher) Lookup(name string) (string, error) {
	l.lookups = append(l.lookups, name)
	return l.path, nil
}

func (l *recordingLauncher) Run(spec launch.Spec) (int, error) {
	l.specs = append(l.specs, spec)
	return l.exitCode, nil
}

// runUseCLI는 실행 경계와 부모 환경까지 값으로 채운 Deps로 새 커맨드 트리를 구성해 실행한다.
func runUseCLI(t *testing.T, layout profile.Layout, launcher launch.Launcher, baseEnv []string, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	root := cli.NewRootCommand(cli.Deps{
		Layout:   layout,
		Stdout:   &outBuf,
		Stderr:   &errBuf,
		Platform: launch.NewPlatform(),
		BaseEnv:  baseEnv,
		Launcher: launcher,
	})
	root.SetArgs(args)
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// envValues는 환경 목록에서 name에 정확히 일치하는 항목의 값을 순서대로 뽑는다. 개수까지
// 돌려주므로 "한 번만 들어 있다"를 셀 수 있다.
func envValues(env []string, name string) []string {
	var out []string
	for _, entry := range env {
		if k, v, ok := strings.Cut(entry, "="); ok && k == name {
			out = append(out, v)
		}
	}
	return out
}

// listedDir은 list 출력에서 name 행의 DIR 값을 읽는다. 주입되는 경로가 사용자가 실제로 본
// 경로와 같은 문자열인지 보려면 비교 대상을 list 출력에서 가져와야 한다.
func listedDir(t *testing.T, layout profile.Layout, name string) string {
	t.Helper()
	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	for _, r := range dataRows(t, stdout) {
		if r[0] == name {
			return r[1]
		}
	}
	t.Fatalf("list output has no %q row", name)
	return ""
}

// 등록된 이름으로 실행하면 조회로 확정된 경로에 "--" 뒤 값이 그대로 전달되고, 자식 환경에는
// 프로필 디렉토리가 CLAUDE_CONFIG_DIR로 한 번만 들어가며 호출 환경에 남아 있던 두 변수는
// 자식에 닿지 않는다.
func TestUse_InjectsProfileConfigDirAndForwardsArgs(t *testing.T) {
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

	stdout, stderr, err := runUseCLI(t, layout, launcher, baseEnv, "use", "work", "--", "--model", "opus")
	if err != nil {
		t.Fatalf("use error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	// 성공 경로에서 ccswitch 자신은 한 줄도 내지 않는다 — 화면이 claude를 직접 실행한 것과
	// 달라지면 안 된다.
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
	if want := []string{"--model", "opus"}; !slices.Equal(spec.Args, want) {
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

// runDefaultUse는 default 대상 실행 한 번을 돌려 자식에게 넘어간 환경 목록을 돌려준다. 프로필을
// 하나도 만들지 않고 실행하므로, 이 경로가 등록 확인을 거치지 않는다는 것도 함께 드러난다.
func runDefaultUse(t *testing.T, baseEnv []string, argv ...string) []string {
	t.Helper()
	launcher := &recordingLauncher{path: fakeExecPath}

	stdout, stderr, err := runUseCLI(t, newTestLayout(t), launcher, baseEnv, argv...)
	if err != nil {
		t.Fatalf("%v error = %v, stderr = %q", argv, err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Fatalf("%v ExitCode() = %d, want 0", argv, code)
	}
	if stdout != "" || stderr != "" {
		t.Fatalf("%v stdout = %q, stderr = %q, want both empty", argv, stdout, stderr)
	}
	if len(launcher.specs) != 1 {
		t.Fatalf("%v got %d launches, want 1", argv, len(launcher.specs))
	}
	return launcher.specs[0].Env
}

// 이름을 생략한 use와 use default는 같은 대상으로 실행되고, 자식 환경에 설정 디렉토리를 가리키는
// 두 변수가 남지 않는다 — 호출 환경에 이미 들어 있어도 그렇다.
func TestUse_DefaultTargetLeavesNoConfigDirEnv(t *testing.T) {
	baseEnv := []string{
		"PATH=/usr/bin",
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
	}

	forms := []struct {
		label string
		env   []string
	}{
		{label: "name omitted", env: runDefaultUse(t, baseEnv, "use")},
		{label: "default by name", env: runDefaultUse(t, baseEnv, "use", "default")},
	}

	for _, form := range forms {
		for _, envName := range []string{"CLAUDE_CONFIG_DIR", "CLAUDE_SECURESTORAGE_CONFIG_DIR"} {
			if got := envValues(form.env, envName); len(got) != 0 {
				t.Errorf("%s: %s values = %q, want none", form.label, envName, got)
			}
		}
		if got := envValues(form.env, "PATH"); !slices.Equal(got, []string{"/usr/bin"}) {
			t.Errorf("%s: PATH values = %q, want unrelated variables untouched", form.label, got)
		}
	}

	// 두 형태가 같은 대상이라는 것은 자식 환경으로만 드러난다 — 같은 부모 환경을 주었으므로
	// 결과 목록도 같아야 한다.
	if !slices.Equal(forms[0].env, forms[1].env) {
		t.Errorf("child env differs between forms:\n%s = %q\n%s = %q",
			forms[0].label, forms[0].env, forms[1].label, forms[1].env)
	}
}

// 기본 설정 디렉토리가 아직 없는 홈에서도 default 대상 실행은 실패하지 않고, ccswitch가 그
// 디렉토리를 만들지도 않는다.
func TestUse_DefaultTargetDoesNotCreateDefaultDir(t *testing.T) {
	for _, argv := range [][]string{{"use"}, {"use", "default"}} {
		t.Run(strings.Join(argv, " "), func(t *testing.T) {
			layout := newTestLayout(t)
			defaultDir := layout.DefaultDir()
			if _, err := os.Stat(defaultDir); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Stat(%q) error = %v, want it absent before the run", defaultDir, err)
			}

			launcher := &recordingLauncher{path: fakeExecPath}
			if _, stderr, err := runUseCLI(t, layout, launcher, nil, argv...); err != nil {
				t.Fatalf("%v error = %v, stderr = %q", argv, err, stderr)
			}
			if len(launcher.specs) != 1 {
				t.Fatalf("%v got %d launches, want 1", argv, len(launcher.specs))
			}

			if _, err := os.Stat(defaultDir); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Stat(%q) error = %v, want it still absent after the run", defaultDir, err)
			}
		})
	}
}

// "--" 뒤의 값은 ccswitch가 해석하지 않고 순서와 개수 그대로 넘어간다.
func TestUse_ForwardsArgumentsAfterDashDashVerbatim(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "no forwarded arguments",
			argv: []string{"use", "work"},
			want: nil,
		},
		{
			name: "flag with value",
			argv: []string{"use", "work", "--", "--model", "opus"},
			want: []string{"--model", "opus"},
		},
		{
			name: "valueless flag",
			argv: []string{"use", "work", "--", "--verbose"},
			want: []string{"--verbose"},
		},
		{
			// 두 번째 "--"는 구분자로 다시 소비되지 않고 값으로 남아야 한다.
			name: "second dash dash kept as a value",
			argv: []string{"use", "work", "--", "--", "-p", "hello"},
			want: []string{"--", "-p", "hello"},
		},
		{
			name: "empty string argument",
			argv: []string{"use", "work", "--", "--model", ""},
			want: []string{"--model", ""},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			layout := newTestLayout(t)
			if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
				t.Fatalf("add error = %v, stderr = %q", err, stderr)
			}

			launcher := &recordingLauncher{path: fakeExecPath}
			_, stderr, err := runUseCLI(t, layout, launcher, nil, tc.argv...)
			if err != nil {
				t.Fatalf("use error = %v, stderr = %q", err, stderr)
			}
			if len(launcher.specs) != 1 {
				t.Fatalf("got %d launches, want 1", len(launcher.specs))
			}
			if got := launcher.specs[0].Args; !slices.Equal(got, tc.want) {
				t.Errorf("Spec.Args = %q, want %q", got, tc.want)
			}
		})
	}
}

// "--" 앞에 이름이 둘 이상 오거나 "--" 없이 알 수 없는 옵션이 오면 실행에 닿기 전에 거부되고,
// 두 메시지 모두 옵션을 어디에 두어야 하는지 알려준다.
func TestUse_UsageErrorsRejectBeforeLaunching(t *testing.T) {
	cases := map[string][]string{
		"more than one profile name": {"use", "work", "personal"},
		"unknown flag without --":    {"use", "-abc"},
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
			if !strings.Contains(stderr, `"--"`) {
				t.Errorf("stderr = %q, want it to point at %q", stderr, "--")
			}
			if len(launcher.lookups) != 0 || len(launcher.specs) != 0 {
				t.Errorf("launcher was called: lookups=%q specs=%v", launcher.lookups, launcher.specs)
			}
		})
	}
}
