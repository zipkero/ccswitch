package cli

import (
	"errors"
	"fmt"

	"github.com/zipkero/ccswitch/internal/profile"
)

// ErrUsage는 cobra/pflag가 인자·플래그 파싱 단계에서 낸 오류를 감싼다. 파싱 동작 자체(POSIX
// 관례, "--" 뒤 위치 인자 등)는 바꾸지 않고, 이미 거부된 결과를 ANALYSIS §3의 종료 코드 2로
// 분류하기 위한 표식으로만 쓴다.
var ErrUsage = errors.New("usage error")

// ErrProfileUnusable은 등록된 프로필의 디렉토리를 그대로 쓸 수 없을 때 돌아온다. 디렉토리가
// 없는 경우와 디렉토리가 아닌 것이 놓인 경우를 한 코드로 묶는다 — 둘 다 "이 프로필로는 실행할
// 수 없다"는 같은 분류이고, 서로 다른 조치는 메시지가 나른다 (D8). profile 패키지가 아니라
// 여기에 있는 것은 이 판정이 profile.Inspect의 상태를 실행 거부로 옮기는 CLI의 정책이기
// 때문이다 — profile은 상태를 계산할 뿐 무엇을 거부할지 모른다.
var ErrProfileUnusable = errors.New("profile cannot be used")

// ErrExecNotFound는 PATH에서 Claude Code 실행 파일을 찾지 못했을 때 돌아온다. 실행 경계가
// 돌려준 exec.ErrNotFound를 이 sentinel로 옮기는 것은 CLI의 정책이다 — launch는 조회가
// 실패했다는 사실만 알고 그것이 어느 종료 코드가 되는지 모른다. 1(그 밖의 실패)로 묶지 않는
// 것은 사용자가 취할 조치가 다르기 때문이다: 여기서는 Claude Code를 설치하거나 PATH를 고친다
// (D8).
var ErrExecNotFound = errors.New("executable not found")

// ErrClaudeAuthFailed는 Claude Code에 위임한 claude auth 명령이 답을 주지 못했을 때 돌아온다.
// logout 실패, rm의 정리 실패, list --accounts의 계정 조회 실패가 모두 이 코드를 쓴다 — 세
// 자리 모두 같은 원인(위임한 명령이 답을 주지 못함)의 다른 발생 지점이기 때문이다
// (profile-auth D12).
var ErrClaudeAuthFailed = errors.New("claude auth command did not answer")

// childExitError는 자식 프로세스가 0이 아닌 코드로 끝났다는 사실만 나른다. ccswitch 자신의
// 실패가 아니므로 이 error가 흐르는 경로는 stderr에 아무것도 덧붙이지 않는다 — Ctrl+C로 세션을
// 빠져나올 때마다 한 줄이 붙으면 화면이 claude를 직접 실행한 것과 달라진다 (D7).
type childExitError struct {
	code int
}

// Error는 사용자에게 보이는 통로가 아니다. 이 문구가 화면에 나왔다면 위 규칙이 이미 깨진 것이다.
func (e *childExitError) Error() string {
	return fmt.Sprintf("claude exited with code %d", e.code)
}

// ExitCode는 도메인 sentinel error를 종료 코드로 옮긴다. 코드 표는 ANALYSIS §3을 따르며,
// 아직 이 표면에 없는 명령이 쓰는 sentinel은 그 명령이 추가될 때 분기도 함께 늘어난다.
func ExitCode(err error) int {
	// 자식이 한 번 뜬 뒤에는 그 종료 코드가 아래 표와 무관하게 그대로 나간다. 표보다 먼저 보는
	// 것이 계약이다 — 자식이 낸 3을 "프로필을 찾을 수 없다"로 다시 해석하면 안 된다 (D7). 한
	// 번의 실행에서 거부와 중계가 겹칠 수 없으므로 두 뜻이 부딪히지 않는다.
	var childErr *childExitError
	if errors.As(err, &childErr) {
		return childErr.code
	}

	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrUsage),
		errors.Is(err, profile.ErrInvalidName),
		errors.Is(err, profile.ErrReservedName):
		return 2
	case errors.Is(err, profile.ErrNotFound):
		return 3
	case errors.Is(err, profile.ErrAlreadyExists):
		return 4
	case errors.Is(err, profile.ErrDirOccupied):
		return 5
	case errors.Is(err, ErrProfileUnusable):
		return 6
	case errors.Is(err, ErrExecNotFound):
		return 7
	case errors.Is(err, ErrClaudeAuthFailed):
		return 8
	case errors.Is(err, profile.ErrStoreCorrupt):
		return 1
	default:
		return 1
	}
}
