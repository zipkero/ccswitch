package cli_test

import (
	"fmt"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// 자식이 끝낸 코드가 ccswitch의 종료 코드로 그대로 나오고, 0이 아닌 코드에서도 ccswitch가
// 아무 메시지도 덧붙이지 않는다. use와 같은 중계 규칙이 login에도 그대로 적용된다(D3의
// 관계 — 자식이 뜬 뒤에는 login도 profile-launcher D7을 그대로 탄다).
func TestLogin_RelaysChildExitCode(t *testing.T) {
	// 130은 ccswitch의 코드 표(0~7) 밖의 값이라, 코드가 표를 거치지 않고 나온다는 것이 값만으로
	// 드러난다. 1·2는 표와 겹치는 값이므로 재해석되지 않는지 함께 본다.
	for _, want := range []int{0, 1, 2, 130} {
		t.Run(fmt.Sprintf("child exit %d", want), func(t *testing.T) {
			layout := newTestLayout(t)
			if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
				t.Fatalf("add error = %v, stderr = %q", err, stderr)
			}

			launcher := &recordingLauncher{path: fakeExecPath, exitCode: want}
			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "login", "work")

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
