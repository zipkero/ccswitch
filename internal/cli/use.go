package cli

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

// claudeExecutable은 PATH에서 찾는 실행 파일 이름이다. 확장자를 붙이지 않는다 — Windows에서는
// 조회 쪽이 PATHEXT를 함께 본다.
const claudeExecutable = "claude"

func newUseCommand(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [name] [-- args...]",
		Short: "Run Claude Code with a profile's configuration directory",
		// 자식 종료 코드를 나르는 error에는 사용자에게 보일 메시지가 없어야 하므로 프레임워크의
		// 기본 오류 출력을 끄고, use의 실패 메시지는 reportUseError가 직접 낸다 (D7). 이 설정은
		// use에만 걸리므로 나머지 세 커맨드의 출력 경로는 그대로다. 대신 use에서 나가는 모든
		// error가 reportUseError를 지나야 한다 — 빠뜨린 경로의 실패는 화면에 아무것도 남기지
		// 않는다.
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			return reportUseError(cmd, useArgs(cmd, args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, forwarded := splitUseArgs(cmd.ArgsLenAtDash(), args)
			return reportUseError(cmd, runUse(deps, name, forwarded))
		},
	}
	// 알 수 없는 옵션을 거부하는 일은 파서에 그대로 맡기고(D11), 메시지에 한 줄만 덧붙인다 —
	// 기본 메시지만으로는 claude에 줄 옵션을 어디에 두어야 하는지 알 수 없다. ErrUsage로 감싸
	// 종료 코드 2 분류는 다른 커맨드와 같게 유지한다.
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return reportUseError(cmd, fmt.Errorf(
			`%w: %v; put options meant for claude after "--"`, ErrUsage, err,
		))
	})
	return cmd
}

// reportUseError는 use의 실패를 stderr에 한 줄로 내고 error를 그대로 흘려보낸다. 접두사와 출력
// 스트림을 cobra에서 그대로 가져오는 것은, use만 기본 오류 출력을 끄기 때문에 메시지 모양이
// 다른 커맨드와 조용히 갈라질 수 있기 때문이다.
func reportUseError(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	// 자식 종료 코드 중계 경로다. ccswitch가 거부한 것이 아니므로 여기에는 한 줄도 붙이지 않는다.
	var childErr *childExitError
	if errors.As(err, &childErr) {
		return err
	}
	cmd.PrintErrln(cmd.ErrPrefix(), err.Error())
	return err
}

// useArgs는 "--" 앞 위치 인자가 0개나 1개인지 본다. "--" 뒤는 개수를 따지지 않는다.
func useArgs(cmd *cobra.Command, args []string) error {
	if n := positionalCount(cmd.ArgsLenAtDash(), args); n > 1 {
		return fmt.Errorf(
			`%w: accepts at most 1 profile name, received %d; put arguments meant for claude after "--"`,
			ErrUsage, n,
		)
	}
	return nil
}

// splitUseArgs는 프로필 이름과 자식에게 그대로 넘길 인자를 가른다. "--" 앞이 비어 있으면
// 이름은 빈 문자열이다.
func splitUseArgs(dashPos int, args []string) (name string, forwarded []string) {
	n := positionalCount(dashPos, args)
	if n > 0 {
		name = args[0]
	}
	return name, args[n:]
}

// positionalCount는 파서가 남긴 "--" 앞 위치 인자 개수를 읽는다. 이 구분을 직접 만들지 않는
// 이유는 파서가 이미 POSIX 관례대로 "--"에서 파싱을 끝내고 그 앞의 개수를 기록해 두기 때문이다
// (D11). "--"가 없었으면 -1이 오고, 그때는 위치 인자 전부가 ccswitch의 것이다.
func positionalCount(dashPos int, args []string) int {
	if dashPos < 0 {
		return len(args)
	}
	return dashPos
}

func runUse(deps Deps, name string, args []string) error {
	dir, err := useTargetDir(deps, name)
	if err != nil {
		return err
	}

	// 조회는 이름 형식·등록 여부·디렉토리 상태 판정 뒤에 온다 (D14). 앞으로 옮기면 Claude Code가
	// 설치되지 않은 환경에서 어떤 이름을 넣어도 같은 오류가 나와, 이름을 잘못 쓴 사용자가 그
	// 사실을 알 수 없다.
	path, err := deps.Launcher.Lookup(claudeExecutable)
	if err != nil {
		// 찾지 못한 경우만 코드 7로 옮기고, 조회가 내는 그 밖의 오류(찾았지만 실행 권한이 없는
		// 경우 등)는 기존 분류에 맡긴다. 감싼 원래 오류의 문구는 싣지 않는다 — 같은 사실을
		// 반복하면서 한 줄이 길어질 뿐이다.
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf(
				`%w: %q is not in PATH; install Claude Code or add the directory containing it to PATH`,
				ErrExecNotFound, claudeExecutable,
			)
		}
		return err
	}
	// 조회가 돌려준 경로는 형태를 따지지 않고 그대로 실행 대상으로 쓴다 (D15). .exe를 먼저
	// 찾거나 shim을 거부하는 규칙을 넣으면 "PATH 조회 결과를 쓴다"고 못 박은 SPEC §4의 범위를
	// 넘어선다.

	// 성공 경로에서 use는 자기 출력을 내지 않는다. add·rm처럼 결과 한 줄을 남기면 화면이
	// claude를 직접 실행한 것과 달라진다.
	code, err := deps.Launcher.Run(launch.Spec{
		Path: path,
		Args: args,
		Env:  deps.Platform.Environ(deps.BaseEnv, dir),
	})
	if err != nil {
		return err
	}
	// 0이 아닌 종료 코드는 실패가 아니라 중계 대상이다. error로 나르는 것은 커맨드에서 종료
	// 코드까지 가는 통로를 하나로 두기 위한 것이고, 그 값은 ccswitch의 코드 표를 거치지 않는다.
	if code != 0 {
		return &childExitError{code: code}
	}
	return nil
}

// useTargetDir는 실행 대상 설정 디렉토리를 정한다. 빈 문자열은 기본 설정 디렉토리를 뜻하며,
// 환경 계산이 변수를 붙이지 않고 제거만 하게 만든다.
//
// 대상 해석이 이름 검증보다 앞에 있어야 한다 — default는 list가 한 행으로 보여주는 실재하는
// 항목이므로, 사용자가 목록에서 본 이름을 그대로 입력했을 때 예약 이름 오류로 거부되면 안 된다.
// 예약 이름 규칙은 add·rm이 기본 설정 디렉토리를 만들거나 지우지 못하게 하려는 것이고 use에는
// 그 위험이 없다 (D12).
func useTargetDir(deps Deps, name string) (string, error) {
	// default 대상은 등록 확인도 디렉토리 상태 확인도 거치지 않는다. 아무것도 주입하지 않으므로
	// Claude Code가 자기 기본 경로를 스스로 정하고, 그 디렉토리가 아직 없어도 세션이 만들 것이라
	// 없다는 사실이 실패가 아니다. ccswitch가 미리 만들지도 않는다.
	if name == "" || name == profile.DefaultName {
		return "", nil
	}
	// 이름 검증이 경로 계산보다 먼저다 — ProfileDir은 이름을 경로에 그대로 잇기 때문에,
	// 검증을 거치지 않은 이름이 홈 밖을 가리키는 경로를 만들 수 있다.
	if err := profile.ValidateName(name); err != nil {
		return "", err
	}

	// 등록 확인이 디렉토리 상태 판정보다 앞이다(D14) — 등록되지 않은 이름의 자리에 우연히
	// 디렉토리가 있어도(예: 이전 add가 디렉토리 생성 뒤 저장에 실패해 남긴 것) 그것을 실행
	// 대상으로 삼지 않는다.
	store, err := profile.Load(deps.Layout.MetadataPath())
	if err != nil {
		return "", err
	}
	if !store.Has(name) {
		return "", fmt.Errorf(
			`%w: %q; run "ccswitch list" to see registered profiles`, profile.ErrNotFound, name,
		)
	}

	dir := deps.Layout.ProfileDir(name)
	// 상태 판정을 새로 만들지 않고 profile.Inspect에 그대로 맡긴다 — list가 missing·unusable로
	// 보여준 것과 use가 거부하는 것이 같은 판정이어야 사용자가 두 출력을 연결할 수 있다. 어느
	// 상태도 고쳐 쓰지 않는다: 없는 디렉토리를 만들어 진행하면 자격증명이 사라진 프로필이 새 빈
	// 프로필처럼 조용히 되살아나, 사용자는 로그인이 풀린 이유를 알 수 없다 (D13).
	switch profile.Inspect(dir) {
	case profile.StatusOK:
		return dir, nil
	case profile.StatusMissing:
		return "", fmt.Errorf(
			`%w: %q is registered but %s does not exist; remove the registration with "ccswitch rm %s" and create it again with "ccswitch add %s"`,
			ErrProfileUnusable, name, dir, name, name,
		)
	default:
		// StatusUnusable이 여기로 온다. 그 자리에 놓인 것을 ccswitch가 치우지 않으므로 조치는
		// 사용자에게 넘기며, 등록은 그대로 두라고 안내한다 — 이름이 이미 등록되어 있어 다시
		// add할 수는 없다. profile이 나중에 상태를 더해도 실행을 거부하는 쪽으로 남는다.
		return "", fmt.Errorf(
			`%w: %q is registered but %s is not a usable directory; move or remove whatever is at that path`,
			ErrProfileUnusable, name, dir,
		)
	}
}
