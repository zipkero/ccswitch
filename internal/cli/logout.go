package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/auth"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newLogoutCommand(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "logout <name>",
		Short: "Sign a profile out of Claude Code",
		Args:  logoutArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogout(deps, args[0])
		},
	}
}

// logoutArgs는 위치 인자가 정확히 1개인지 본다. claude auth logout에는 옵션이 없으므로(D5)
// login과 달리 "--" 뒤 전달을 받지 않는다 — "--" 뒤에 무엇이 와도 거부한다.
func logoutArgs(cmd *cobra.Command, args []string) error {
	dashPos := cmd.ArgsLenAtDash()
	if dashPos >= 0 && len(args) > dashPos {
		return fmt.Errorf(`%w: logout does not take arguments after "--"`, ErrUsage)
	}
	if n := positionalCount(dashPos, args); n != 1 {
		return fmt.Errorf(`%w: accepts exactly 1 profile name, received %d`, ErrUsage, n)
	}
	return nil
}

func runLogout(deps Deps, name string) error {
	dir, err := resolveTargetDir(deps, name)
	if err != nil {
		return err
	}

	// 조회는 대상 해석 뒤에 온다 — 순서는 runUse·runLogin과 같다(D14).
	path, err := lookupClaudeExecutable(deps)
	if err != nil {
		return err
	}

	captured, captureErr := deps.Launcher.Capture(launch.Spec{
		Path: path,
		Args: auth.LogoutArgs(),
		Env:  deps.Platform.Environ(deps.BaseEnv, dir),
	})
	if captureErr == nil {
		captureErr = auth.ReadLogout(captured.Stdout, captured.Stderr, captured.ExitCode)
	}
	if captureErr != nil {
		// 위임한 명령이 답을 주지 못한 것이지 프로필의 로그인 여부 문제가 아니므로, 상태를
		// 바꾸는 명령(예: login)을 조치로 권하지 않는다 — list의 조회 실패 메시지와 같은 기준이다.
		return fmt.Errorf(
			"%w: %v; check that %q is a working Claude Code installation, or reinstall it",
			ErrClaudeAuthFailed, captureErr, path,
		)
	}

	displayDir := dir
	if name == profile.DefaultName {
		displayDir = deps.Layout.DefaultDir()
	}
	fmt.Fprintf(deps.Stderr, "Logged out profile %q at %s\n", name, displayDir)
	return nil
}
