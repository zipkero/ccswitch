// Package profile은 프로필의 존재 여부·위치·이름에 대한 단일 기준을 소유한다.
// 파일시스템과 사용자 설정 위치를 제외한 어떤 외부와도 이야기하지 않으며, 종료 코드나
// 출력 형식을 알지 못한다.
package profile

import (
	"os"
	"strings"
)

// DefaultName은 Claude Code의 기본 설정 디렉토리를 가리키는 예약 이름이다. 메타데이터에
// 저장되지 않고 조회 시점에 합성된다.
const DefaultName = "default"

// Layout은 경로 계산의 기준값이다. 홈 루트와 사용자 설정 루트만 담고, 프로필 디렉토리·기본
// 설정 디렉토리·메타데이터 파일 경로는 모두 이 값에서 순수 계산으로 나온다. 플랫폼에 따라
// 갈리는 부분은 NewLayout 한 곳뿐이며, 나머지 계산은 runtime.GOOS를 보지 않는다.
type Layout struct {
	Home       string
	ConfigRoot string
}

// NewLayout은 표준 라이브러리의 사용자 홈·설정 위치 판정에 맡겨 Layout을 만든다.
func NewLayout() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, err
	}
	configRoot, err := os.UserConfigDir()
	if err != nil {
		return Layout{}, err
	}
	return Layout{Home: home, ConfigRoot: configRoot}, nil
}

// ProfileDir은 이름으로 구분되는 프로필 디렉토리 경로를 돌려준다. 경로는 등록 정보에 저장되지
// 않고 항상 이 함수로 다시 계산된다.
func (l Layout) ProfileDir(name string) string {
	return joinNative(l.Home, ".claude-"+name)
}

// DefaultDir은 Claude Code의 기본 설정 디렉토리 경로다.
func (l Layout) DefaultDir() string {
	return joinNative(l.Home, ".claude")
}

// MetadataPath는 프로필 등록 정보 파일의 경로다.
func (l Layout) MetadataPath() string {
	return joinNative(l.ConfigRoot, "ccswitch", "profiles.json")
}

// joinNative는 base에 이미 쓰인 구분자를 보고 그 구분자로 세그먼트를 잇는다 — base에
// 백슬래시가 있으면 백슬래시로, 없으면 슬래시로 잇는다. filepath.Join은 컴파일 대상 OS의
// 구분자로 결과를 다시 쓰므로 Windows에서 macOS 스타일 base를 넣으면 구분자가 뒤섞인
// 문자열이 나온다. joinNative는 runtime.GOOS를 보지 않고 base 문자열만 보므로, 실제 Windows
// 실행에서는 os.UserHomeDir()이 준 백슬래시가 그대로 이어지고, macOS 스타일 base를 넣는
// 테스트에서도 같은 코드가 슬래시로 이어진 결과를 낸다.
func joinNative(base string, segs ...string) string {
	sep := "/"
	if strings.ContainsRune(base, '\\') {
		sep = "\\"
	}
	parts := append([]string{strings.TrimRight(base, `/\`)}, segs...)
	return strings.Join(parts, sep)
}
