package launch

import (
	"runtime"
	"strings"
)

// 자식 환경에서 반드시 지우는 환경변수 이름. 이 둘만이 프로필 사이의 격리를 깬다 — 어느
// 프로필로 실행해도 같은 자격증명·설정 파일을 보게 만든다. API 키류는 모든 프로필에 똑같이
// 작용하므로 목록에 넣지 않는다 (D4).
const (
	configDirEnv              = "CLAUDE_CONFIG_DIR"
	secureStorageConfigDirEnv = "CLAUDE_SECURESTORAGE_CONFIG_DIR"
)

// Platform은 플랫폼에 따라 갈리는 실행 환경 사실만 담는다. 이 값을 받는 계산은 어느 환경에서
// 돌려도 같은 결과를 내므로, Windows에서 Unix 정책을, Unix에서 Windows 정책을 각각 테스트할
// 수 있다.
type Platform struct {
	// EnvNamesCaseInsensitive는 환경변수 이름을 비교할 때 대소문자를 무시할지를 정한다.
	// Windows에서는 셸이 만들어 둔 claude_config_dir도 같은 변수이므로 무시해야 하고,
	// Unix에서는 실제로 다른 변수이므로 구분해야 한다 (D5).
	EnvNamesCaseInsensitive bool
}

// NewPlatform은 실행 중인 플랫폼에 맞는 Platform을 만든다. 이 패키지에서 runtime.GOOS를 보는
// 유일한 자리다.
func NewPlatform() Platform {
	return Platform{EnvNamesCaseInsensitive: runtime.GOOS == "windows"}
}

// Environ은 base에서 설정 디렉토리를 가리키는 환경변수를 모두 지우고, configDir이 비어 있지
// 않으면 CLAUDE_CONFIG_DIR을 그 값으로 붙인 새 목록을 돌려준다. base는 바꾸지 않는다.
//
// configDir이 빈 문자열이면 기본 설정 디렉토리를 뜻하며 아무것도 붙이지 않는다 — 기본 경로를
// 값으로 넣으면 Claude Code가 평소와 다른 자격증명 항목과 다른 설정 파일 위치를 보게 된다.
// configDir은 받은 문자열 그대로 쓴다. 절대 경로화·Clean·뒤 구분자 손질을 하지 않는 것은,
// macOS가 이 문자열의 해시로 Keychain 항목을 가르므로 같은 프로필이 실행마다 다른 문자열을
// 받으면 로그아웃된 것처럼 보이기 때문이다 (D6).
func (p Platform) Environ(base []string, configDir string) []string {
	out := make([]string, 0, len(base)+1)
	for _, entry := range base {
		if p.removed(entry) {
			continue
		}
		out = append(out, entry)
	}
	if configDir != "" {
		out = append(out, configDirEnv+"="+configDir)
	}
	return out
}

// removed는 "NAME=VALUE" 항목 하나가 제거 대상인지 본다. "="가 없는 항목은 이름을 가릴 수
// 없으므로 건드리지 않는다 — Windows 환경 블록에는 "=C:=C:\dir"처럼 이름이 비어 있는 항목이
// 섞여 들어온다.
func (p Platform) removed(entry string) bool {
	name, _, ok := strings.Cut(entry, "=")
	if !ok {
		return false
	}
	for _, target := range [...]string{configDirEnv, secureStorageConfigDirEnv} {
		if name == target {
			return true
		}
		if p.EnvNamesCaseInsensitive && strings.EqualFold(name, target) {
			return true
		}
	}
	return false
}
