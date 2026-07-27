package profile

import (
	"errors"
	"fmt"
	"regexp"
)

// ErrInvalidName은 이름이 형식 규칙을 어겼을 때 돌아온다.
var ErrInvalidName = errors.New("invalid profile name")

// ErrReservedName은 예약된 이름(DefaultName)으로 프로필을 만들려고 할 때 돌아온다.
var ErrReservedName = errors.New("profile name is reserved")

// nameFormat은 허용되는 이름 형식이다. 대소문자 구분 없는 파일시스템에서도 이름과 디렉토리가
// 일대일이 되도록 소문자로 고정한다(D7).
var nameFormat = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}$`)

// nameFormatHint는 형식 위반 메시지에 그대로 실어 사용자에게 허용 형식을 한 줄로 보여준다.
const nameFormatHint = "profile names must start with a lowercase letter or digit, " +
	"contain only lowercase letters, digits, '_', or '-', and be 1-32 characters long"

// ValidateName은 name을 프로필 이름으로 쓸 수 있는지 검사한다. 형식 위반과 예약 이름을
// ErrInvalidName/ErrReservedName으로 구분해 돌려준다.
func ValidateName(name string) error {
	if name == DefaultName {
		return &nameError{
			msg: fmt.Sprintf("%q is reserved and cannot be used as a profile name", name),
			err: ErrReservedName,
		}
	}
	if !nameFormat.MatchString(name) {
		return &nameError{
			msg: fmt.Sprintf("%q is not a valid profile name: %s", name, nameFormatHint),
			err: ErrInvalidName,
		}
	}
	return nil
}

// nameError는 사용자에게 보이는 메시지(msg)와 errors.Is 분류용 sentinel(err)을 나눠 갖는다.
// %w로 sentinel을 감싸면 sentinel 자신의 문구가 메시지 끝에 그대로 붙어 같은 내용이 두 번
// 나오므로, Unwrap만으로 sentinel을 연결하고 메시지에는 다시 넣지 않는다.
type nameError struct {
	msg string
	err error
}

func (e *nameError) Error() string { return e.msg }
func (e *nameError) Unwrap() error { return e.err }
