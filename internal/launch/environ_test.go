package launch_test

import (
	"runtime"
	"slices"
	"testing"

	"github.com/zipkero/ccswitch/internal/launch"
)

// 두 정책 값이 모두 여기서 만들어지므로, 실행 플랫폼이 무엇이든 아래 테스트가 두 정책을
// 동시에 검증한다.
var bothPolicies = []launch.Platform{
	{EnvNamesCaseInsensitive: false},
	{EnvNamesCaseInsensitive: true},
}

func policyName(p launch.Platform) string {
	if p.EnvNamesCaseInsensitive {
		return "case-insensitive names"
	}
	return "case-sensitive names"
}

// 이름의 대소문자가 걸리지 않는 케이스는 두 정책에서 결과가 같아야 한다.
func TestPlatformEnviron(t *testing.T) {
	cases := []struct {
		name      string
		base      []string
		configDir string
		want      []string
	}{
		{
			name:      "관련 없는 변수는 그대로 두고 대상 디렉토리를 뒤에 붙인다",
			base:      []string{"PATH=/usr/bin", "HOME=/home/u"},
			configDir: "/home/u/.claude-work",
			want:      []string{"PATH=/usr/bin", "HOME=/home/u", "CLAUDE_CONFIG_DIR=/home/u/.claude-work"},
		},
		{
			name: "호출 환경에 이미 있던 두 변수는 사라지고 새 값이 한 번만 남는다",
			base: []string{
				"CLAUDE_CONFIG_DIR=/stale",
				"PATH=/usr/bin",
				"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
			},
			configDir: "/home/u/.claude-work",
			want:      []string{"PATH=/usr/bin", "CLAUDE_CONFIG_DIR=/home/u/.claude-work"},
		},
		{
			name: "대상이 비어 있으면 제거만 일어난다",
			base: []string{
				"CLAUDE_CONFIG_DIR=/stale",
				"PATH=/usr/bin",
				"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
			},
			configDir: "",
			want:      []string{"PATH=/usr/bin"},
		},
		{
			// Windows 환경 블록에는 "=C:=C:\dir"처럼 이름이 비어 있는 항목이 섞여 들어오고,
			// 값 구분자가 아예 없는 항목도 목록에 있을 수 있다. 둘 다 건드리지 않는다.
			name:      "이름을 가릴 수 없는 항목은 그대로 둔다",
			base:      []string{`=C:=C:\work`, "CLAUDE_CONFIG_DIR"},
			configDir: "",
			want:      []string{`=C:=C:\work`, "CLAUDE_CONFIG_DIR"},
		},
		{
			name:      "주입하는 경로 문자열은 가공하지 않는다",
			base:      []string{"PATH=/usr/bin"},
			configDir: `C:\Users\u\.claude-work\`,
			want:      []string{"PATH=/usr/bin", `CLAUDE_CONFIG_DIR=C:\Users\u\.claude-work\`},
		},
	}

	for _, p := range bothPolicies {
		t.Run(policyName(p), func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					got := p.Environ(tc.base, tc.configDir)
					if !slices.Equal(got, tc.want) {
						t.Errorf("Environ() = %q, want %q", got, tc.want)
					}
				})
			}
		})
	}
}

// 이름의 대소문자가 걸리는 케이스는 정책에 따라 결과가 갈려야 한다. 무시 정책에서 소문자
// 이름이 남으면 "프로필을 지정했는데 다른 계정이 보인다"로 나타나고, 구분 정책에서 소문자
// 이름이 사라지면 Claude Code가 읽지도 않는 변수를 조용히 없애는 것이 된다.
func TestPlatformEnviron_EnvNameCasePolicy(t *testing.T) {
	base := []string{
		"claude_config_dir=/stale",
		"Claude_SecureStorage_Config_Dir=/stale-secure",
		"PATH=/usr/bin",
	}
	const configDir = "/home/u/.claude-work"

	cases := []struct {
		platform launch.Platform
		want     []string
	}{
		{
			platform: launch.Platform{EnvNamesCaseInsensitive: false},
			want: []string{
				"claude_config_dir=/stale",
				"Claude_SecureStorage_Config_Dir=/stale-secure",
				"PATH=/usr/bin",
				"CLAUDE_CONFIG_DIR=" + configDir,
			},
		},
		{
			platform: launch.Platform{EnvNamesCaseInsensitive: true},
			want: []string{
				"PATH=/usr/bin",
				"CLAUDE_CONFIG_DIR=" + configDir,
			},
		},
	}

	for _, tc := range cases {
		t.Run(policyName(tc.platform), func(t *testing.T) {
			got := tc.platform.Environ(base, configDir)
			if !slices.Equal(got, tc.want) {
				t.Errorf("Environ() = %q, want %q", got, tc.want)
			}
		})
	}
}

// 호출자가 넘긴 목록은 부모 환경 그 자체일 수 있으므로 Environ이 제자리에서 고치면 안 된다.
func TestPlatformEnviron_DoesNotMutateBase(t *testing.T) {
	for _, p := range bothPolicies {
		t.Run(policyName(p), func(t *testing.T) {
			base := []string{"CLAUDE_CONFIG_DIR=/stale", "PATH=/usr/bin"}
			before := slices.Clone(base)

			p.Environ(base, "/home/u/.claude-work")

			if !slices.Equal(base, before) {
				t.Errorf("base = %q, want it unchanged %q", base, before)
			}
		})
	}
}

func TestNewPlatform_CaseInsensitiveOnlyOnWindows(t *testing.T) {
	want := runtime.GOOS == "windows"
	if got := launch.NewPlatform().EnvNamesCaseInsensitive; got != want {
		t.Errorf("NewPlatform().EnvNamesCaseInsensitive = %v, want %v on %s", got, want, runtime.GOOS)
	}
}
