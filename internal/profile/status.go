package profile

import (
	"errors"
	"os"
)

// Status는 등록 정보와 실제 디렉토리가 조회 시점에 어떤 관계인지를 나타낸다. 저장되지
// 않고 매번 Inspect로 계산된다.
type Status int

const (
	// StatusOK는 경로에 디렉토리가 있는 정상 상태다.
	StatusOK Status = iota
	// StatusMissing은 경로에 아무것도 없는 상태다 — 사용자가 디렉토리를 밖에서 지웠거나
	// 등록 저장 뒤 디렉토리 생성이 실패한 경우다.
	StatusMissing
	// StatusUnusable은 경로에 디렉토리가 아닌 것이 있거나 조사할 수 없는 상태다.
	StatusUnusable
)

func (s Status) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusMissing:
		return "missing"
	case StatusUnusable:
		return "unusable"
	default:
		return "unknown"
	}
}

// Inspect는 dir의 실제 상태를 조사한다. 어떤 상태도 고쳐 쓰지 않는다.
func Inspect(dir string) Status {
	info, err := os.Stat(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StatusMissing
		}
		// 권한 문제 등 존재 여부를 판정할 수 없는 경우도 조치가 필요하다는 점에서
		// 디렉토리가 아닌 것이 놓인 경우와 같이 묶는다.
		return StatusUnusable
	}
	if !info.IsDir() {
		return StatusUnusable
	}
	return StatusOK
}
