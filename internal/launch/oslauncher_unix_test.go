//go:build !windows

package launch

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// 자식 역할을 하는 테스트 바이너리 자신이 어떤 동작을 할지 고르는 환경변수. 비어 있으면
// TestHelperProcess는 아무것도 하지 않는다.
const (
	helperModeEnv = "CCSWITCH_LAUNCH_HELPER"
	helperOutEnv  = "CCSWITCH_LAUNCH_HELPER_OUT"
)

// TestHelperProcess는 직접 실행되는 테스트가 아니라 Run이 띄우는 자식이다. Run은 자식의
// 표준 출력을 부모 콘솔에 넘기므로 결과는 파일로 돌려준다.
func TestHelperProcess(t *testing.T) {
	switch os.Getenv(helperModeEnv) {
	case "report-sigint":
		// Go 런타임은 시작 시점에 이미 무시 상태였던 시그널을 Ignored로 보고한다.
		got := "default"
		if signal.Ignored(os.Interrupt) {
			got = "ignored"
		}
		os.WriteFile(os.Getenv(helperOutEnv), []byte(got), 0o644)
		os.Exit(0)
	case "interrupt-parent":
		syscall.Kill(os.Getppid(), syscall.SIGINT)
		// 부모가 인터럽트를 처리할 시간을 두고 끝난다. 부모가 그 사이 죽었다면 Run은 돌아오지 않는다.
		time.Sleep(200 * time.Millisecond)
		os.Exit(0)
	}
}

func helperSpec(t *testing.T, mode string, extraEnv ...string) Spec {
	t.Helper()
	env := append(os.Environ(), helperModeEnv+"="+mode)
	return Spec{
		Path: os.Args[0],
		Args: []string{"-test.run=^TestHelperProcess$"},
		Env:  append(env, extraEnv...),
	}
}

// 대기 중 인터럽트를 삼키는 장치가 자식에게 새면 claude가 SIGINT를 무시하는 채로 시작한다.
// Unix에서 SIG_IGN은 exec를 넘어 상속되므로 signal.Ignore로 구현하면 이 테스트가 실패한다.
func TestOSLauncherRun_ChildReceivesDefaultSIGINT(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sigint")
	code, err := OSLauncher{}.Run(helperSpec(t, "report-sigint", helperOutEnv+"="+out))
	if err != nil || code != 0 {
		t.Fatalf("Run() = %d, %v; want 0, nil", code, err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", out, err)
	}
	if s := strings.TrimSpace(string(got)); s != "default" {
		t.Errorf("child SIGINT disposition = %q, want %q", s, "default")
	}
}

// 자식이 도는 동안 부모에게 온 인터럽트는 부모를 끝내지 않아야 한다(D9). 이 보장이 깨지면
// 테스트 프로세스 자체가 SIGINT로 죽는다.
func TestOSLauncherRun_ParentSurvivesInterruptWhileWaiting(t *testing.T) {
	code, err := OSLauncher{}.Run(helperSpec(t, "interrupt-parent"))
	if err != nil || code != 0 {
		t.Fatalf("Run() = %d, %v; want 0, nil", code, err)
	}
}
