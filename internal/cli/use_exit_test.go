package cli_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// sharedErrPrefix는 use가 아닌 커맨드가 실제로 낸 실패 한 줄에서 접두사를 뽑는다. 접두사를
// 상수로 박아 두면 use만 기본 오류 출력을 끈 탓에 두 경로가 갈라져도 테스트가 놓친다.
func sharedErrPrefix(t *testing.T) string {
	t.Helper()
	_, stderr, err := runCommand(t, newTestLayout(t), "add", "Work")
	if err == nil {
		t.Fatalf(`add "Work" error = nil, want an invalid name error`)
	}
	prefix, _, ok := strings.Cut(strings.TrimRight(stderr, "\n"), " ")
	if !ok {
		t.Fatalf("add stderr = %q, want a prefixed message line", stderr)
	}
	return prefix
}

// 자식이 끝낸 코드가 ccswitch의 종료 코드로 그대로 나오고, 0이 아닌 코드에서도 ccswitch가
// 아무 메시지도 덧붙이지 않는다.
func TestUse_RelaysChildExitCode(t *testing.T) {
	// 130은 ccswitch의 코드 표(0~7) 밖의 값이라, 코드가 표를 거치지 않고 나온다는 것이 값만으로
	// 드러난다. 1·2·3은 표와 겹치는 값이므로 재해석되지 않는지 함께 본다.
	for _, want := range []int{0, 1, 2, 130} {
		t.Run(fmt.Sprintf("child exit %d", want), func(t *testing.T) {
			layout := newTestLayout(t)
			if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
				t.Fatalf("add error = %v, stderr = %q", err, stderr)
			}

			launcher := &recordingLauncher{path: fakeExecPath, exitCode: want}
			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "use", "work")

			if got := cli.ExitCode(err); got != want {
				t.Errorf("ExitCode() = %d, want %d", got, want)
			}
			if (err != nil) != (want != 0) {
				t.Errorf("error = %v, want it non-nil only for a non-zero child code", err)
			}
			if stdout != "" || stderr != "" {
				t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
			}
			if len(launcher.specs) != 1 {
				t.Errorf("got %d launches, want 1", len(launcher.specs))
			}
		})
	}
}

// 실행을 거부한 경우에는 다른 커맨드와 같은 접두사가 붙은 한 줄만 stderr에 남고, 사용법 전문은
// 나오지 않는다. 파싱 단계 거부와 실행 단계 거부가 각각 다른 자리를 지나므로 둘을 함께 본다.
func TestUse_RejectionPrintsOnePrefixedLine(t *testing.T) {
	wantPrefix := sharedErrPrefix(t)

	cases := map[string][]string{
		"more than one profile name": {"use", "work", "personal"},
		"unknown flag without --":    {"use", "-abc"},
		"invalid profile name":       {"use", "Work"},
	}

	for name, argv := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			launcher := &recordingLauncher{path: fakeExecPath}

			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, argv...)
			if err == nil {
				t.Fatalf("%v error = nil, want a rejection", argv)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if strings.Contains(stderr, "Usage:") {
				t.Errorf("stderr = %q, want no usage text", stderr)
			}

			lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
			if len(lines) != 1 {
				t.Fatalf("stderr = %q, want exactly one line", stderr)
			}
			if !strings.HasPrefix(lines[0], wantPrefix+" ") {
				t.Errorf("stderr line = %q, want it to start with %q", lines[0], wantPrefix)
			}

			if len(launcher.lookups) != 0 || len(launcher.specs) != 0 {
				t.Errorf("launcher was called: lookups=%q specs=%v", launcher.lookups, launcher.specs)
			}
		})
	}
}
