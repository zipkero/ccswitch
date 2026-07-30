package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

// 등록되지 않은 이름, 디렉토리를 쓸 수 없는 프로필(missing·occupied), PATH에서 실행 파일을
// 찾지 못한 경우 네 가지에서 use와 login이 같은 이름·같은 상태에 대해 같은 종료 코드로
// 거부되고, stderr 한 줄이 접두사까지 같은 문자열이다(SPEC §5.4) — 둘 다 target.go의 같은
// 판정 함수를 거치기 때문이다.
func TestUseAndLogin_RejectWithSameMessageAndCode(t *testing.T) {
	wantPrefix := sharedErrPrefix(t)

	cases := []struct {
		label    string
		setup    func(t *testing.T, layout profile.Layout) string
		wantCode int
		launcher func() launch.Launcher
	}{
		{
			label:    "unregistered name",
			setup:    func(t *testing.T, layout profile.Layout) string { return "work" },
			wantCode: 3,
			launcher: func() launch.Launcher { return &recordingLauncher{path: fakeExecPath} },
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
			launcher: func() launch.Launcher { return &recordingLauncher{path: fakeExecPath} },
		},
		{
			label: "file at the profile path",
			setup: func(t *testing.T, layout profile.Layout) string {
				registerWithoutDir(t, layout, "work")
				if err := os.WriteFile(layout.ProfileDir("work"), []byte(occupyingContent), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				return "work"
			},
			wantCode: 6,
			launcher: func() launch.Launcher { return &recordingLauncher{path: fakeExecPath} },
		},
		{
			label: "executable not found",
			setup: func(t *testing.T, layout profile.Layout) string {
				if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
					t.Fatalf("add error = %v, stderr = %q", err, stderr)
				}
				return "work"
			},
			wantCode: 7,
			launcher: func() launch.Launcher { return &notFoundLauncher{} },
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			layout := newTestLayout(t)
			name := tc.setup(t, layout)

			wantList := listSnapshot(t, layout)
			wantStore := storeSnapshot(t, layout)

			_, useStderr, useErr := runUseCLI(t, layout, tc.launcher(), nil, "use", name)
			if useErr == nil {
				t.Fatalf("use %q error = nil, want a rejection", name)
			}
			if got := cli.ExitCode(useErr); got != tc.wantCode {
				t.Errorf("use ExitCode() = %d, want %d", got, tc.wantCode)
			}

			_, loginStderr, loginErr := runUseCLI(t, layout, tc.launcher(), nil, "login", name)
			if loginErr == nil {
				t.Fatalf("login %q error = nil, want a rejection", name)
			}
			if got := cli.ExitCode(loginErr); got != tc.wantCode {
				t.Errorf("login ExitCode() = %d, want %d", got, tc.wantCode)
			}

			if useStderr != loginStderr {
				t.Errorf("stderr differs:\nuse   = %q\nlogin = %q", useStderr, loginStderr)
			}
			if !strings.HasPrefix(useStderr, wantPrefix+" ") {
				t.Errorf("stderr = %q, want it to start with %q", useStderr, wantPrefix)
			}

			// 거부 경로는 등록 파일도 list 출력도 실행 전후로 그대로다.
			if got := storeSnapshot(t, layout); got != wantStore {
				t.Errorf("store file changed:\nbefore = %q\nafter  = %q", wantStore, got)
			}
			if got := listSnapshot(t, layout); got != wantList {
				t.Errorf("list output changed:\nbefore = %q\nafter  = %q", wantList, got)
			}
		})
	}
}
