package cli_test

import (
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// cobra/pflag가 인자·플래그 파싱 단계에서 내는 오류도 ANALYSIS §3의 "잘못된 사용법"과 같은
// 종료 코드 2로 분류되어야 한다.
func TestUsageErrors_MapToExitCode2(t *testing.T) {
	cases := map[string][]string{
		"add missing name argument":   {"add"},
		"add unknown flag without --": {"add", "-abc"},
		"list unexpected argument":    {"list", "extra"},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)
			_, _, err := runCommand(t, layout, args...)
			if err == nil {
				t.Fatalf("%v error = nil, want usage error", args)
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
		})
	}
}
