package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// nameRejectionCases는 SPEC §5.2·§5.3이 요구하는 거부 대상 이름이다. "-"로 시작하는 값은
// SPEC §3의 POSIX 관례에 따라 "--" 뒤에 두고 넘겨야 add까지 도달한다(그 앞은 인자 파서 소관).
var nameRejectionCases = map[string]struct {
	input      string
	wantSubstr string
}{
	"uppercase":      {"Work", "lowercase"},
	"space":          {"my profile", "lowercase"},
	"path separator": {"a/b", "lowercase"},
	"dot-dot":        {"..", "lowercase"},
	"empty":          {"", "lowercase"},
	"too long":       {strings.Repeat("a", 33), "lowercase"},
	"leading dash":   {"-abc", "lowercase"},
	"default":        {"default", "reserved"},
}

// 이름 검증이 디스크 접근보다 먼저 끝나야 하므로, 각 케이스에서 홈 아래에 새 항목이 생기지
// 않고 등록 파일도 생기지 않아야 한다.
func TestAdd_RejectsFormatViolationsAndReservedNameWithoutTouchingDisk(t *testing.T) {
	for name, tc := range nameRejectionCases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)

			_, stderr, err := runCommand(t, layout, "add", "--", tc.input)
			if err == nil {
				t.Fatalf("add %q error = nil, want rejection", tc.input)
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
			if !strings.Contains(stderr, tc.wantSubstr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.wantSubstr)
			}

			homeEntries, readErr := os.ReadDir(layout.Home)
			if readErr != nil {
				t.Fatalf("ReadDir(Home) error = %v", readErr)
			}
			if len(homeEntries) != 0 {
				t.Errorf("Home has %d entries after rejected add, want 0: %v", len(homeEntries), homeEntries)
			}

			if _, statErr := os.Stat(layout.MetadataPath()); !os.IsNotExist(statErr) {
				t.Errorf("MetadataPath() stat error = %v, want it to not exist", statErr)
			}
		})
	}
}

// 등록 파일이 이미 있는 상태에서 거부되어도 그 바이트가 실행 전후로 그대로인지 확인한다 —
// 이름 검증이 Store.Load보다 앞서더라도, Load 이후 단계에서 실패가 파일을 건드리지 않는다는
// 것은 별도로 보여야 한다.
func TestAdd_RejectsWithoutChangingExistingRegistryFile(t *testing.T) {
	for name, tc := range nameRejectionCases {
		t.Run(name, func(t *testing.T) {
			layout := newTestLayout(t)

			store, err := profile.Load(layout.MetadataPath())
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if err := store.Add("existing"); err != nil {
				t.Fatalf("Add() error = %v", err)
			}
			if err := store.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			before, err := os.ReadFile(layout.MetadataPath())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}

			_, stderr, err := runCommand(t, layout, "add", "--", tc.input)
			if err == nil {
				t.Fatalf("add %q error = nil, want rejection", tc.input)
			}
			if code := cli.ExitCode(err); code != 2 {
				t.Errorf("ExitCode() = %d, want 2", code)
			}
			if !strings.Contains(stderr, tc.wantSubstr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tc.wantSubstr)
			}

			after, err := os.ReadFile(layout.MetadataPath())
			if err != nil {
				t.Fatalf("ReadFile() error = %v", err)
			}
			if string(before) != string(after) {
				t.Errorf("registry file changed after rejected add:\nbefore=%q\nafter=%q", before, after)
			}
		})
	}
}
