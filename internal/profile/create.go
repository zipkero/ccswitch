package profile

import (
	"errors"
	"fmt"
	"os"
)

// ErrDirOccupied는 프로필 디렉토리가 놓일 자리에 비어 있지 않은 디렉토리나 디렉토리가
// 아닌 것이 이미 있을 때 돌아온다.
var ErrDirOccupied = errors.New("target path is occupied")

// EnsureProfileDir은 dir 자리에 프로필 디렉토리를 만든다. 아무것도 없으면 새로 만들고, 빈
// 디렉토리가 이미 있으면 그대로 통과시킨다 — 디렉토리 생성 뒤 등록 저장이 실패해도 같은
// 이름으로 다시 add하면 이 경로로 회복되기 때문이다 (D14). 디렉토리가 아닌 것이나 비어 있지
// 않은 디렉토리가 있으면 안을 읽지도 지우지도 않고 ErrDirOccupied로 실패한다.
func EnsureProfileDir(dir string) error {
	info, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dir, 0o755)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return &dirOccupiedError{
			msg: fmt.Sprintf("%s already exists and is not a directory; remove it or choose a different profile name", dir),
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return &dirOccupiedError{
			msg: fmt.Sprintf("%s already has contents; remove them or choose a different profile name", dir),
		}
	}
	return nil
}

// dirOccupiedError는 사용자에게 보이는 메시지와 errors.Is 분류용 sentinel(ErrDirOccupied)을
// 나눠 갖는다 — name.go의 nameError와 같은 이유로, %w로 감싸면 sentinel 자신의 문구("target
// path is occupied")가 메시지 끝에 다시 붙어 경로·조치 안내와 겹친다.
type dirOccupiedError struct {
	msg string
}

func (e *dirOccupiedError) Error() string { return e.msg }
func (e *dirOccupiedError) Unwrap() error { return ErrDirOccupied }
