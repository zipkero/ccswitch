package cli_test

import (
	"os"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// --skip-logout이면 PATH 조회도 정리 위임도 걸리지 않는다 — 조회가 항상 실패하는 대역을 써서
// 그 대역이 한 번도 불리지 않았음을 확인한다. -y와 함께 줘도 자격증명이 남을 수 있다는 경고는
// 그대로 나온다.
func TestRm_SkipLogoutFlag_DeletesWithoutLookupOrCapture(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	launcher := &notFoundLauncher{}
	_, stderr, err := runRmCLIWithLauncher(t, layout, nil, false, launcher, "rm", "work", "--skip-logout", "--yes")
	if err != nil {
		t.Fatalf("rm error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	if len(launcher.lookups) != 0 {
		t.Errorf("got %d lookups, want 0", len(launcher.lookups))
	}
	if len(launcher.captures) != 0 {
		t.Errorf("got %d captures, want 0", len(launcher.captures))
	}
	if !strings.Contains(stderr, "credentials may remain") {
		t.Errorf("stderr = %q, want a credential warning", stderr)
	}

	if _, statErr := os.Stat(layout.ProfileDir("work")); !os.IsNotExist(statErr) {
		t.Errorf("ProfileDir() stat error = %v, want it to not exist", statErr)
	}
	stdout, _, err := runCommand(t, layout, "list")
	if err != nil {
		t.Fatalf("list error = %v", err)
	}
	for _, r := range dataRows(t, stdout) {
		if r[0] == "work" {
			t.Errorf("list still shows removed profile: %v", r)
		}
	}
}

// 대화형에서 경고는 확인 프롬프트보다 먼저 stderr에 나온다 — 정리를 건너뛴다는 결정이 확정되는
// 자리가 승인을 묻는 자리보다 앞이어야, 사용자가 승인하는 대상에 "자격증명은 남는다"가
// 들어간다.
func TestRm_SkipLogoutFlag_WarningPrecedesPrompt(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	launcher := &notFoundLauncher{}
	_, stderr, err := runRmCLIWithLauncher(t, layout, strings.NewReader("y\n"), true, launcher, "rm", "work", "--skip-logout")
	if err != nil {
		t.Fatalf("rm error = %v, stderr = %q", err, stderr)
	}

	warnIdx := strings.Index(stderr, "credentials may remain")
	promptIdx := strings.Index(stderr, "Remove profile")
	if warnIdx < 0 {
		t.Fatalf("stderr = %q, want a credential warning", stderr)
	}
	if promptIdx < 0 {
		t.Fatalf("stderr = %q, want the confirmation prompt", stderr)
	}
	if warnIdx > promptIdx {
		t.Errorf("warning at %d, prompt at %d; want the warning first", warnIdx, promptIdx)
	}
}

// --skip-logout으로도 거절하면 아무것도 지워지지 않는다 — 경고를 낸 뒤에도 프롬프트의 거부
// 결과는 그대로다.
func TestRm_SkipLogoutFlag_InteractiveDecline_KeepsState(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	dirBefore, err := os.ReadDir(layout.ProfileDir("work"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	registryBefore, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	launcher := &notFoundLauncher{}
	_, stderr, err := runRmCLIWithLauncher(t, layout, strings.NewReader("n\n"), true, launcher, "rm", "work", "--skip-logout")
	if err != nil {
		t.Fatalf("rm error = %v", err)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Errorf("ExitCode() = %d, want 0", code)
	}
	if !strings.Contains(stderr, "Cancelled") {
		t.Errorf("stderr = %q, want it to mention cancellation", stderr)
	}
	if len(launcher.lookups) != 0 || len(launcher.captures) != 0 {
		t.Errorf("launcher was called despite decline: lookups = %v, captures = %v", launcher.lookups, launcher.captures)
	}

	dirAfter, err := os.ReadDir(layout.ProfileDir("work"))
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(dirAfter) != len(dirBefore) {
		t.Errorf("directory entries changed: before=%v after=%v", dirBefore, dirAfter)
	}
	registryAfter, err := os.ReadFile(layout.MetadataPath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(registryBefore) != string(registryAfter) {
		t.Errorf("registry file changed:\nbefore=%q\nafter=%q", registryBefore, registryAfter)
	}
}
