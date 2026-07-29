package cli_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// notFoundLauncher는 조회가 실행 파일을 찾지 못한 상황만 만든다. 실제 구현이 그렇듯
// exec.ErrNotFound를 감싼 *exec.Error를 돌려주므로, CLI가 그 판별에 기대는 계약까지 함께
// 검증된다. 기록은 recordingLauncher에 그대로 맡긴다.
type notFoundLauncher struct {
	recordingLauncher
}

func (l *notFoundLauncher) Lookup(name string) (string, error) {
	l.lookups = append(l.lookups, name)
	return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
}

// 조회 실패가 코드 7이 되고, 같은 조회 실패 상황에서도 사용자가 방금 입력한 값에 대한 판정이
// 먼저 나온다 (D14). 조회를 대역으로 구성하므로 프로세스 전역 PATH를 건드리지 않고, 각 케이스가
// 자기 임시 홈만 쓰므로 병렬로 돌아도 서로 간섭하지 않는다.
func TestUse_LookupFailureRejectsAfterNameAndStatusChecks(t *testing.T) {
	t.Parallel()

	wantPrefix := sharedErrPrefix(t)

	cases := []struct {
		label string
		// setup은 실행 전 상태를 꾸미고 use에 줄 이름을 돌려준다.
		setup func(t *testing.T, layout profile.Layout) string
		// wantCode는 ANALYSIS §3의 코드 표에서 이 거부에 해당하는 값이다.
		wantCode int
		// wantLookup은 조회에 닿았는지다. 앞선 판정에 걸린 케이스에서 false여야 판정 순서가
		// 고정된다 — 조회가 항상 실패하는 대역이므로, 닿았다면 코드가 7로 덮였을 것이다.
		wantLookup bool
		// wantInStderr는 메시지가 무엇을 찾지 못했는지와 어디를 확인해야 하는지를 나르는지 본다.
		wantInStderr []string
	}{
		{
			label: "usable profile",
			setup: func(t *testing.T, layout profile.Layout) string {
				if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
					t.Fatalf("add error = %v, stderr = %q", err, stderr)
				}
				return "work"
			},
			wantCode:     7,
			wantLookup:   true,
			wantInStderr: []string{`"claude"`, "PATH"},
		},
		{
			label: "unregistered name",
			setup: func(t *testing.T, layout profile.Layout) string {
				return "work"
			},
			wantCode: 3,
		},
		{
			label: "registered but directory deleted",
			setup: func(t *testing.T, layout profile.Layout) string {
				if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
					t.Fatalf("add error = %v, stderr = %q", err, stderr)
				}
				if err := os.RemoveAll(layout.ProfileDir("work")); err != nil {
					t.Fatalf("RemoveAll() error = %v", err)
				}
				return "work"
			},
			wantCode: 6,
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			// 조회 실패를 대역으로 구성한다는 것은 프로세스 전역 PATH가 그대로 남는다는 뜻이다.
			// PATH를 바꿔 실패를 만들면 병렬로 도는 다른 테스트까지 끌려간다 (D1).
			wantPath := os.Getenv("PATH")

			layout := newTestLayout(t)
			name := tc.setup(t, layout)

			launcher := &notFoundLauncher{}
			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "use", name)

			if err == nil {
				t.Fatalf("use %q error = nil, want a rejection", name)
			}
			if got := cli.ExitCode(err); got != tc.wantCode {
				t.Errorf("ExitCode() = %d, want %d", got, tc.wantCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}

			// 거부는 다른 커맨드와 같은 접두사가 붙은 한 줄로 나온다. use만 프레임워크의 기본
			// 오류 출력을 끄고 있으므로, 새 거부 경로가 그 출력을 지나지 않으면 화면에 아무것도
			// 남지 않는다.
			lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
			if len(lines) != 1 || lines[0] == "" {
				t.Fatalf("stderr = %q, want exactly one line", stderr)
			}
			if !strings.HasPrefix(lines[0], wantPrefix+" ") {
				t.Errorf("stderr line = %q, want it to start with %q", lines[0], wantPrefix)
			}
			for _, want := range tc.wantInStderr {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr = %q, want it to contain %q", stderr, want)
				}
			}

			if got := len(launcher.lookups) > 0; got != tc.wantLookup {
				t.Errorf("lookups = %q, want lookup reached = %v", launcher.lookups, tc.wantLookup)
			}
			// 어느 케이스에서도 프로세스는 뜨지 않는다.
			if len(launcher.specs) != 0 {
				t.Errorf("launcher ran a process: specs = %v", launcher.specs)
			}

			if got := os.Getenv("PATH"); got != wantPath {
				t.Errorf("process PATH changed:\nbefore = %q\nafter  = %q", wantPath, got)
			}
		})
	}
}
