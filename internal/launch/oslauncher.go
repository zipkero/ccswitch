package launch

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
)

// OSLauncher는 Launcher의 실제 구현이다. 부모 콘솔 인계와 대기 중 인터럽트 무시를 이 구현
// 안에만 둔다 — 무시가 걸려 있는 구간이 자식의 수명과 정확히 같아야 하고, 그 구간을 아는 것은
// 여기뿐이다 (D9).
type OSLauncher struct{}

// Lookup은 PATH 조회를 표준 라이브러리에 그대로 맡긴다. Windows에서는 PATHEXT가 함께
// 적용되므로 실행 파일 이름에 확장자를 붙여 넘기지 않는다.
func (OSLauncher) Lookup(name string) (string, error) {
	return exec.LookPath(name)
}

// Run은 자식을 띄우고 끝날 때까지 기다린 뒤 종료 코드를 돌려준다. 자식이 0이 아닌 코드로
// 끝난 것은 error로 보지 않는다.
func (OSLauncher) Run(spec Spec) (int, error) {
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Env = spec.Env
	// 실제 표준 입출력 파일을 그대로 넘긴다 (D10). *os.File을 주면 표준 라이브러리가 파이프도
	// 복사 goroutine도 만들지 않고 핸들을 그대로 자식에게 넘기므로, Claude Code의 TUI 판정이
	// 비대화형으로 떨어지지 않는다. CLI 경계의 스트림은 테스트에서 버퍼로 갈아끼우는 값이라
	// 자식에게 줄 콘솔 핸들이 될 수 없다.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Dir과 SysProcAttr을 일부러 채우지 않는다. 자식은 부모의 현재 디렉토리를 물려받아야 같은
	// 프로젝트를 보고, Windows에서 CreationFlags를 직접 넣지 않으면 CREATE_NEW_CONSOLE이 붙지
	// 않아 부모 콘솔을 공유한다 — 새 창을 띄우지 않는 동작은 "아무것도 하지 않는 것"으로
	// 실현되므로, 여기서 무엇을 더하면 그대로 깨진다 (D2).

	// 같은 콘솔에 붙은 프로세스는 Ctrl+C를 함께 받는다. 무시를 걸지 않으면 부모가 먼저 끝나
	// 자식이 남는다. 대상을 인터럽트 하나로 한정하는 것은, 종료 시그널까지 무시하면 대기 중
	// ccswitch를 정상 수단으로 끝낼 수 없기 때문이다 (D9).
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return 0, err
	}
	return 0, nil
}
