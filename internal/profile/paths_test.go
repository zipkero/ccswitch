package profile

import "testing"

// 경로 계산 함수는 runtime.GOOS를 보지 않는 순수 함수여야 하므로, 이 테스트를 실행하는
// 플랫폼과 무관하게 macOS 스타일 기준값을 넣어도 같은 문자열이 나와야 한다 (ANALYSIS D6).
func TestLayoutPaths_PlatformIndependent(t *testing.T) {
	l := Layout{
		Home:       "/Users/me",
		ConfigRoot: "/Users/me/Library/Application Support",
	}

	if got, want := l.ProfileDir("work"), "/Users/me/.claude-work"; got != want {
		t.Errorf("ProfileDir(%q) = %q, want %q", "work", got, want)
	}
	if got, want := l.DefaultDir(), "/Users/me/.claude"; got != want {
		t.Errorf("DefaultDir() = %q, want %q", got, want)
	}
	if got, want := l.MetadataPath(), "/Users/me/Library/Application Support/ccswitch/profiles.json"; got != want {
		t.Errorf("MetadataPath() = %q, want %q", got, want)
	}
}

func TestLayoutPaths_WindowsStyleHome(t *testing.T) {
	l := Layout{
		Home:       `C:\Users\me`,
		ConfigRoot: `C:\Users\me\AppData\Roaming`,
	}

	if got, want := l.ProfileDir("work"), `C:\Users\me\.claude-work`; got != want {
		t.Errorf("ProfileDir(%q) = %q, want %q", "work", got, want)
	}
	if got, want := l.DefaultDir(), `C:\Users\me\.claude`; got != want {
		t.Errorf("DefaultDir() = %q, want %q", got, want)
	}
	if got, want := l.MetadataPath(), `C:\Users\me\AppData\Roaming\ccswitch\profiles.json`; got != want {
		t.Errorf("MetadataPath() = %q, want %q", got, want)
	}
}
