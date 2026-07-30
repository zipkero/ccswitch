package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/auth"
	"github.com/zipkero/ccswitch/internal/launch"
)

func newLoginCommand(deps Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login <name> [-- args...]",
		Short: "Start Claude Code's login procedure for a profile",
		// login도 자식 종료 코드를 그대로 중계하므로 use와 같은 이유로 기본 오류 출력을 끈다
		// (D7). 실패 메시지는 reportChildRelayError가 직접 낸다.
		SilenceErrors: true,
		Args: func(cmd *cobra.Command, args []string) error {
			return reportChildRelayError(cmd, loginArgs(cmd, args))
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name, forwarded := splitPositionalArgs(cmd.ArgsLenAtDash(), args)
			return reportChildRelayError(cmd, runLogin(deps, name, forwarded))
		},
	}
	setForwardingFlagErrorFunc(cmd)
	return cmd
}

// loginArgs는 "--" 앞 위치 인자가 정확히 1개인지 본다. login은 use와 달리 이름 생략을
// 허용하지 않는다 — 생략하면 오타 한 번으로 기본 설정 디렉토리의 로그인 절차가 시작되고,
// 되돌리려면 브라우저 OAuth를 다시 거쳐야 한다(D5).
func loginArgs(cmd *cobra.Command, args []string) error {
	if n := positionalCount(cmd.ArgsLenAtDash(), args); n != 1 {
		return fmt.Errorf(
			`%w: accepts exactly 1 profile name, received %d; put arguments meant for claude after "--"`,
			ErrUsage, n,
		)
	}
	return nil
}

func runLogin(deps Deps, name string, args []string) error {
	dir, err := resolveTargetDir(deps, name)
	if err != nil {
		return err
	}

	// 조회는 대상 해석 뒤에 온다 — 순서는 runUse와 같다(D14).
	path, err := lookupClaudeExecutable(deps)
	if err != nil {
		return err
	}

	// 성공 경로에서 login은 자기 출력을 내지 않는다 — 화면에 남는 것은 claude auth login이
	// 낸 것뿐이다. "--" 뒤 값은 해석하지 않고 auth.LoginArgs가 만든 인자 뒤에 그대로 붙는다.
	code, err := deps.Launcher.Run(launch.Spec{
		Path: path,
		Args: auth.LoginArgs(args),
		Env:  deps.Platform.Environ(deps.BaseEnv, dir),
	})
	if err != nil {
		return err
	}
	if code != 0 {
		return &childExitError{code: code}
	}
	return nil
}
