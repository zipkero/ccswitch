package cli

import (
	"errors"

	"github.com/zipkero/ccswitch/internal/profile"
)

// ErrUsage는 cobra/pflag가 인자·플래그 파싱 단계에서 낸 오류를 감싼다. 파싱 동작 자체(POSIX
// 관례, "--" 뒤 위치 인자 등)는 바꾸지 않고, 이미 거부된 결과를 ANALYSIS §3의 종료 코드 2로
// 분류하기 위한 표식으로만 쓴다.
var ErrUsage = errors.New("usage error")

// ExitCode는 도메인 sentinel error를 종료 코드로 옮긴다. 코드 표는 ANALYSIS §3을 따르며,
// 아직 이 표면에 없는 명령이 쓰는 sentinel은 그 명령이 추가될 때 분기도 함께 늘어난다.
func ExitCode(err error) int {
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
	case errors.Is(err, profile.ErrStoreCorrupt):
		return 1
	default:
		return 1
	}
}
