// Package cli는 인자 파싱, 서브커맨드 배치, 사람이 읽는 출력, 도메인 error를 메시지와
// 종료 코드로 옮기는 일을 소유한다. 도메인 규칙을 다시 판단하지 않는다.
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

// Deps는 커맨드 트리 구성에 필요한 의존성이다. 전역 상태를 두지 않아 테스트가 매번 새
// 값으로 갈아끼우고 새 커맨드 트리를 만들 수 있다.
//
// Interactive는 표준 입력이 실제 터미널인지를 나타낸다. 이 경계가 os.Stdin을 직접 들여다보면
// 승인·거부·비대화형 세 갈래를 테스트로 구성할 수 없으므로, 진입점이 판정한 값을 그대로
// 받는다(D13).
//
// Platform·BaseEnv·Launcher도 같은 성격이다 — 실행 환경에서 온 값을 진입점이 한 번 만들어
// 값으로 넘기고, 테스트가 같은 자리에 자기 값을 넣는다. BaseEnv를 값으로 받기 때문에 자식
// 환경 계산이 프로세스 전역 환경변수를 건드리지 않고 검증된다.
type Deps struct {
	Layout      profile.Layout
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Interactive bool

	Platform launch.Platform
	BaseEnv  []string
	Launcher launch.Launcher
}

// NewRootCommand는 Deps를 값으로 받아 커맨드 트리를 구성해 돌려준다.
func NewRootCommand(deps Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "ccswitch",
		Short: "Manage isolated Claude Code profiles",
		// 실행 시점 오류에는 사용법을 함께 뱉지 않도록 누르지만, 오류 메시지 자체는
		// cobra의 기본 출력에 맡긴다 — 이 함수가 stderr로 SetErr한 스트림에 나간다 (D12).
		SilenceUsage: true,
	}
	root.SetOut(deps.Stdout)
	root.SetErr(deps.Stderr)
	// pflag가 알 수 없는 플래그를 만나면 내는 오류를 ErrUsage로 감싼다 — 어떤 값이 플래그로
	// 오인되는지는 그대로 두고(예: "--" 없는 "-abc"), 그 결과를 종료 코드 2로 분류만 한다.
	root.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	})

	root.AddCommand(newAddCommand(deps))
	root.AddCommand(newListCommand(deps))
	root.AddCommand(newLoginCommand(deps))
	root.AddCommand(newLogoutCommand(deps))
	root.AddCommand(newRmCommand(deps))
	root.AddCommand(newUseCommand(deps))

	return root
}

// exactArgs는 cobra.ExactArgs(n)을 감싸 인자 개수 오류를 ErrUsage로 분류한다.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			return fmt.Errorf("%w: %v", ErrUsage, err)
		}
		return nil
	}
}

// noArgs는 cobra.NoArgs를 감싸 인자 개수 오류를 ErrUsage로 분류한다.
func noArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	return nil
}
