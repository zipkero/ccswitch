package cli_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
)

// 실제 프로세스를 한 번 왕복해, 판정 계층(auth.ReadStatus의 순수 테스트)이 실제 캡처 실행
// 경로에서도 성립하는지 본다(ANALYSIS D13) — 자식이 표준 출력에 loggedIn을 담은 JSON을 내고
// 0이 아닌 코드로 끝나게 하고, 부모가 그것을 답으로 읽어 그 행을 계정으로 표시하는지 확인한다.
// use_roundtrip_test.go의 TestMain·realRunLauncher 틀을 그대로 쓴다.
func TestListAccounts_RealProcessRoundTrip(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}

	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	recordPath := filepath.Join(t.TempDir(), "child.json")
	const wantJSON = `{"loggedIn": true, "authMethod": "claudeai", "apiProvider": "firstParty", "email": "roundtrip@example.com", "subscriptionType": "team"}`
	// claude auth status가 로그인 상태에서도 0이 아닌 코드로 끝날 수 있다는 것이 판정에서
	// 무시되는 값이라는 사실(D4)을 실제 프로세스 경로에서도 확인한다.
	const wantExitCode = 3

	baseEnv := append(os.Environ(),
		childRecordEnv+"="+recordPath,
		childExitEnv+"="+strconv.Itoa(wantExitCode),
		childStdoutEnv+"="+wantJSON,
	)

	stdout, stderr, err := runUseCLI(t, layout, realRunLauncher{path: self}, baseEnv, "list", "--accounts")
	if err != nil {
		t.Fatalf("list --accounts error = %v, stderr = %q", err, stderr)
	}
	if code := cli.ExitCode(err); code != 0 {
		t.Fatalf("ExitCode() = %d, want 0 (a query answer, not a failure)", code)
	}

	rows := accountRows(t, stdout)
	var found bool
	for _, r := range rows {
		if r[0] != "work" {
			continue
		}
		found = true
		if r[3] != "roundtrip@example.com" || r[4] != "team" {
			t.Errorf("row = %v, want ACCOUNT=%q PLAN=%q", r, "roundtrip@example.com", "team")
		}
	}
	if !found {
		t.Fatalf(`list --accounts output missing "work" row: %v`, rows)
	}
}
