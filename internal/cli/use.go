package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/launch"
)

func newUseCommand(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use [name] [-- args...]",
		Short: "Run Claude Code with a profile's configuration directory",
		// 자식 종료 코드를 나르는 error에는 사용자에게 보일 메시지가 없어야 하므로 프레임워크의
		// 기본 오류 출력을 끄고, use의 실패 메시지는 reportChildRelayError가 직접 낸다 (D7). 이
		// 설정은 use·login에만 걸리므로 나머지 커맨드의 출력 경로는 그대로다. 대신 use에서
		// 나가는 모든 error가 reportChildRelayError를 지나야 한다 — 빠뜨린 경로의 실패는
		// 화면에 아무것도 남기지 않는다.
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			return reportChildRelayError(cmd, useArgs(cmd, args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, forwarded := splitPositionalArgs(cmd.ArgsLenAtDash(), args)
			return reportChildRelayError(cmd, runUse(deps, name, forwarded))
		},
	}
	setForwardingFlagErrorFunc(cmd)
	return cmd
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

func runUse(deps Deps, name string, args []string) error {
	dir, err := resolveTargetDir(deps, name)
	if err != nil {
		return err
	}

	// 조회는 이름 형식·등록 여부·디렉토리 상태 판정 뒤에 온다 (D14). 앞으로 옮기면 Claude Code가
	// 설치되지 않은 환경에서 어떤 이름을 넣어도 같은 오류가 나와, 이름을 잘못 쓴 사용자가 그
	// 사실을 알 수 없다.
	path, err := lookupClaudeExecutable(deps)
	if err != nil {
		return err
	}

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
