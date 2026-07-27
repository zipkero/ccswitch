// ccswitch는 계정별 Claude Code 설정 프로필을 만들고 확인하고 지우는 CLI다.
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, isInteractive(os.Stdin)))
}

// isInteractive는 표준 입력이 실제 터미널(문자 장치)인지를 본다(D13). 이 판정은 진입점에서만
// 하고, 결과는 값으로 CLI 경계에 넘긴다 — CLI 경계가 os.Stdin을 직접 보면 승인·거부·비대화형
// 세 갈래를 테스트로 구성할 수 없다.
func isInteractive(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// run은 프로세스 종료를 모르는 나머지 코드와 os.Exit를 잇는 유일한 자리다.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer, interactive bool) int {
	layout, err := profile.NewLayout()
	if err != nil {
		fmt.Fprintln(stderr, "ccswitch:", err)
		return 1
	}

	root := cli.NewRootCommand(cli.Deps{
		Layout:      layout,
		Stdin:       stdin,
		Stdout:      stdout,
		Stderr:      stderr,
		Interactive: interactive,
	})
	root.SetArgs(args)
	err = root.Execute()
	return cli.ExitCode(err)
}
