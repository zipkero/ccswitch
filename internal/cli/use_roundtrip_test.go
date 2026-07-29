package cli_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/launch"
)

// 자식 모드를 켜는 환경변수. Platform.Environ의 제거 대상이 아니므로 부모 환경에 넣어 두면
// 자식까지 그대로 도달한다.
const (
	childRecordEnv = "CCSWITCH_TEST_CHILD_RECORD"
	childExitEnv   = "CCSWITCH_TEST_CHILD_EXIT"
)

// childExitSetupFailed는 자식이 기록을 남기지 못했을 때 쓰는 코드다. 왕복이 깨졌을 때 "중계가
// 틀렸다"와 "자식 쪽 준비가 틀렸다"를 종료 코드만으로 구별하려고 요청 코드와 겹치지 않게 둔다.
const childExitSetupFailed = 90

// childRecord는 자식이 실제로 받은 인자와 환경을 부모에게 넘기는 형식이다.
type childRecord struct {
	Args []string `json:"args"`
	Env  []string `json:"env"`
}

// TestMain은 이 테스트 바이너리를 자식 프로세스 역할로도 쓴다. 왕복 검증이 조회 결과 실행
// 파일로 바이너리 자신을 지정하므로, testing의 플래그 파싱보다 먼저 자식 모드를 가려내야
// 한다 — 자식은 "--model opus"처럼 testing이 모르는 인자를 받는다.
func TestMain(m *testing.M) {
	if recordPath := os.Getenv(childRecordEnv); recordPath != "" {
		os.Exit(runAsChild(recordPath, os.Getenv(childExitEnv), os.Args[1:]))
	}
	os.Exit(m.Run())
}

func runAsChild(recordPath, requestedCode string, args []string) int {
	data, err := json.Marshal(childRecord{Args: args, Env: os.Environ()})
	if err != nil {
		fmt.Fprintln(os.Stderr, "child: marshal record:", err)
		return childExitSetupFailed
	}
	if err := os.WriteFile(recordPath, data, 0o600); err != nil {
		fmt.Fprintln(os.Stderr, "child: write record:", err)
		return childExitSetupFailed
	}
	code, err := strconv.Atoi(requestedCode)
	if err != nil {
		fmt.Fprintln(os.Stderr, "child: requested exit code:", err)
		return childExitSetupFailed
	}
	return code
}

// realRunLauncher는 조회만 대역으로 두고 실행은 OSLauncher에 그대로 맡긴다. 조회 결과를 값으로
// 지정하므로 프로세스 전역 PATH를 건드리지 않고도 실제 실행 경로를 지날 수 있다.
type realRunLauncher struct {
	path string
}

func (l realRunLauncher) Lookup(string) (string, error) { return l.path, nil }

func (realRunLauncher) Run(spec launch.Spec) (int, error) { return launch.OSLauncher{}.Run(spec) }

// 실제 프로세스를 한 번 왕복해, 기록용 대역으로는 닿지 않는 OSLauncher 경로에서 환경 주입·인자
// 전달·종료 코드 중계가 함께 성립하는지 본다 (D16).
func TestUse_RealProcessRoundTrip(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("Executable() error = %v", err)
	}

	layout := newTestLayout(t)
	if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
		t.Fatalf("add error = %v, stderr = %q", err, stderr)
	}
	// 비교 대상을 list 출력에서 가져와, 사용자가 본 경로와 자식에게 실제로 닿은 경로가 같은
	// 문자열인지 본다.
	wantDir := listedDir(t, layout, "work")

	recordPath := filepath.Join(t.TempDir(), "child.json")
	// 130은 ccswitch의 코드 표(0~7) 밖의 값이라, 자식의 코드가 표를 거치지 않고 나온다는 것이
	// 값만으로 드러난다.
	const wantCode = 130
	// 실제 부모 환경에서 시작한다 — 자식이 Go 프로그램이므로 플랫폼이 요구하는 항목을 빼면 안
	// 되고, 호출 환경에 남은 두 변수가 실제 실행에서도 사라지는지 함께 보게 된다.
	baseEnv := append(os.Environ(),
		"CLAUDE_CONFIG_DIR=/stale",
		"CLAUDE_SECURESTORAGE_CONFIG_DIR=/stale-secure",
		childRecordEnv+"="+recordPath,
		childExitEnv+"="+strconv.Itoa(wantCode),
	)

	wantArgs := []string{"--model", "opus"}
	argv := append([]string{"use", "work", "--"}, wantArgs...)

	stdout, stderr, err := runUseCLI(t, layout, realRunLauncher{path: self}, baseEnv, argv...)

	if got := cli.ExitCode(err); got != wantCode {
		t.Fatalf("ExitCode() = %d, want %d (error = %v, stderr = %q)", got, wantCode, err, stderr)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
	}

	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", recordPath, err)
	}
	var rec childRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", data, err)
	}

	if !slices.Equal(rec.Args, wantArgs) {
		t.Errorf("child args = %q, want %q", rec.Args, wantArgs)
	}
	if got := envValues(rec.Env, "CLAUDE_CONFIG_DIR"); !slices.Equal(got, []string{wantDir}) {
		t.Errorf("child CLAUDE_CONFIG_DIR = %q, want exactly one %q", got, wantDir)
	}
	if got := envValues(rec.Env, "CLAUDE_SECURESTORAGE_CONFIG_DIR"); len(got) != 0 {
		t.Errorf("child CLAUDE_SECURESTORAGE_CONFIG_DIR = %q, want none", got)
	}
}
