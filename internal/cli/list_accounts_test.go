package cli_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

// accountRows는 `list --accounts`의 NAME·DIR·STATUS·ACCOUNT·PLAN 다섯 열 출력을 헤더를 뺀
// 데이터 행으로 쪼갠다. list_test.go의 dataRows는 옵션 없는 세 열 출력 전용이라(그 파일은
// 고치지 않는다) 다섯 열용은 이 파일이 따로 둔다.
func accountRows(t *testing.T, stdout string) [][]string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	if len(lines) == 0 {
		t.Fatalf("stdout has no lines")
	}
	if fields := strings.Fields(lines[0]); len(fields) != 5 || fields[0] != "NAME" {
		t.Fatalf("unexpected header line: %q", lines[0])
	}
	rows := make([][]string, 0, len(lines)-1)
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) != 5 {
			t.Fatalf("unexpected data line %q: got %d fields", line, len(fields))
		}
		rows = append(rows, fields)
	}
	return rows
}

// accountsScript는 scriptedLauncher.Capture 호출 하나가 돌려줄 값이다.
type accountsScript struct {
	captured launch.Captured
	err      error
}

// scriptedLauncher는 Capture 호출마다 정해 둔 queue의 값을 순서대로 돌려준다. list --accounts는
// profile.StatusOK인 행만 행 순서대로 하나씩 조회하므로(ANALYSIS D8), 이름을 알파벳 순서에 맞춰
// 고르면 queue 순서로 어느 행에 대한 응답인지 예측할 수 있다. Run이 불리면 테스트 자체가
// 잘못된 것이다 — list --accounts는 콘솔을 넘기는 실행에 닿지 않는다.
type scriptedLauncher struct {
	path  string
	queue []accountsScript

	lookups int
	calls   []launch.Spec
}

func (l *scriptedLauncher) Lookup(string) (string, error) {
	l.lookups++
	return l.path, nil
}

func (l *scriptedLauncher) Run(launch.Spec) (int, error) {
	panic("list --accounts must not hand off the console")
}

func (l *scriptedLauncher) Capture(spec launch.Spec) (launch.Captured, error) {
	i := len(l.calls)
	l.calls = append(l.calls, spec)
	if i >= len(l.queue) {
		panic(fmt.Sprintf("unexpected Capture call #%d: %+v", i+1, spec))
	}
	s := l.queue[i]
	return s.captured, s.err
}

// 로그인·로그인 안 됨·조회 실패·상태 불량(default가 미생성이라 missing) 네 행을 한 표에 섞어
// 돌린다. 조회 실패가 있어도 표는 stdout에 그대로 나오고, 실패한 프로필만 stderr에 남으며
// 전체 종료 코드는 8이 된다(SPEC §5.5, §5.11).
func TestList_AccountsShowsFourDistinctRowKinds(t *testing.T) {
	layout := newTestLayout(t)
	for _, name := range []string{"loggedin", "loggedout", "queryfail"} {
		if _, stderr, err := runCommand(t, layout, "add", name); err != nil {
			t.Fatalf("add %q error = %v, stderr = %q", name, err, stderr)
		}
	}
	// default 디렉토리는 만들지 않는다 — missing 상태라 조회하지 않는 네 번째 행이 된다
	// (ANALYSIS D7).

	launcher := &scriptedLauncher{
		path: fakeExecPath,
		queue: []accountsScript{
			{captured: launch.Captured{
				ExitCode: 0,
				Stdout:   `{"loggedIn": true, "authMethod": "claudeai", "apiProvider": "firstParty", "email": "work@example.com", "subscriptionType": "max"}`,
			}},
			{captured: launch.Captured{
				ExitCode: 1,
				Stdout:   `{"loggedIn": false, "authMethod": "none", "apiProvider": "firstParty"}`,
			}},
			{captured: launch.Captured{ExitCode: 0, Stdout: "not json"}},
		},
	}

	stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "list", "--accounts")
	if err == nil {
		t.Fatalf("Execute() error = nil, want a failure (one query could not be read)")
	}
	if code := cli.ExitCode(err); code != 8 {
		t.Errorf("ExitCode() = %d, want 8", code)
	}

	rows := accountRows(t, stdout)
	got := make(map[string][2]string, len(rows))
	for _, r := range rows {
		got[r[0]] = [2]string{r[3], r[4]}
	}
	want := map[string][2]string{
		profile.DefaultName: {"-", "-"},
		"loggedin":          {"work@example.com", "max"},
		"loggedout":         {"logged-out", "-"},
		"queryfail":         {"unknown", "-"},
	}
	for name, wantCols := range want {
		if got[name] != wantCols {
			t.Errorf("row %q ACCOUNT/PLAN = %v, want %v", name, got[name], wantCols)
		}
	}

	if !strings.Contains(stderr, "queryfail") {
		t.Errorf("stderr = %q, want it to mention the failed profile", stderr)
	}
	if launcher.lookups != 1 {
		t.Errorf("lookups = %d, want exactly 1 (PATH is looked up once, not per row)", launcher.lookups)
	}
	if len(launcher.calls) != 3 {
		t.Errorf("Capture calls = %d, want 3 (one per ok row, none for the missing default)", len(launcher.calls))
	}
}

// PATH 조회가 실패하면 표를 내지 않고 코드 7이다 — use・login과 같은 메시지를 재사용한다.
func TestList_AccountsLookupFailureSkipsTableAndQueries(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	launcher := &notFoundLauncher{}
	stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "list", "--accounts")
	if err == nil {
		t.Fatalf("Execute() error = nil, want a lookup failure")
	}
	if code := cli.ExitCode(err); code != 7 {
		t.Errorf("ExitCode() = %d, want 7", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty (no table when the lookup fails)", stdout)
	}
	if stderr == "" {
		t.Errorf("stderr is empty, want a message")
	}
	if len(launcher.captures) != 0 {
		t.Errorf("captures = %v, want none", launcher.captures)
	}
}

// 옵션 없는 list는 이 feature 전후로 출력이 같고(list_test.go가 그대로 통과하는 것으로 이미
// 확인된다), 실행 경계에도 닿지 않는다 — 기록용 대역을 채워도 조회·실행 기록이 0이다
// (SPEC §5.6).
func TestList_WithoutAccountsFlagDoesNotTouchLauncher(t *testing.T) {
	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}

	launcher := &recordingLauncher{path: fakeExecPath}
	stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "list")
	if err != nil {
		t.Fatalf("list error = %v, stderr = %q", err, stderr)
	}

	rows := dataRows(t, stdout)
	if len(rows) != 2 {
		t.Fatalf("got %d data rows, want 2: %v", len(rows), rows)
	}

	if len(launcher.lookups) != 0 || len(launcher.specs) != 0 || len(launcher.captures) != 0 {
		t.Errorf("launcher was called: lookups=%q specs=%v captures=%v", launcher.lookups, launcher.specs, launcher.captures)
	}
}
