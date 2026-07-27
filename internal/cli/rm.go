package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newRmCommand(deps Deps) *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a profile's directory and registration",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(deps, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	return cmd
}

func runRm(deps Deps, name string, yes bool) error {
	// 예약 이름(default)과 형식 위반은 등록 파일을 읽기도 전에 가장 먼저 거부한다 — 프롬프트
	// 와 삭제 어느 쪽에도 도달하지 않는다.
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	store, err := profile.Load(deps.Layout.MetadataPath())
	if err != nil {
		return err
	}

	// 등록 여부를 확인 프롬프트·삭제보다 먼저 본다. 이 순서가 아니면 미등록 이름의 자리에
	// 우연히 디렉토리가 있을 때(예: 이전 add가 디렉토리 생성 뒤 저장에 실패해 남긴 경우)
	// 등록 확인 없이 그 디렉토리를 지워버린다. store.Remove는 !Has일 때 아무것도 바꾸지
	// 않고 ErrNotFound만 돌려주므로 여기서 그대로 재사용한다.
	if !store.Has(name) {
		return store.Remove(name)
	}

	dir := deps.Layout.ProfileDir(name)

	if !yes {
		approved, err := confirmRemoval(deps, name, dir)
		if err != nil {
			return err
		}
		if !approved {
			fmt.Fprintf(deps.Stderr, "Cancelled: %q was not removed\n", name)
			return nil
		}
	}

	// 승인 후에는 디렉토리를 먼저 지우고 등록을 나중에 지운다(D14) — 등록 저장이 실패해도
	// 남는 상태(등록만 있고 디렉토리가 없음)는 list에서 missing으로 보이고 rm 재실행으로
	// 정리된다.
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := store.Remove(name); err != nil {
		return err
	}
	if err := store.Save(); err != nil {
		return err
	}

	fmt.Fprintf(deps.Stderr, "Removed profile %q at %s\n", name, dir)
	return nil
}

// confirmRemoval은 승인 여부를 정한다. 비대화형인데 확인 생략 플래그가 없으면 아무것도
// 지우지 않고 ErrUsage로 실패한다 — "비대화형에서 확인 생략 플래그 누락"은 ANALYSIS §3의
// 종료 코드 2 정의에 이미 들어 있는 경우다. 대화형이면 프롬프트를 띄우고 "y"/"yes"만
// 승인으로 본다.
func confirmRemoval(deps Deps, name, dir string) (bool, error) {
	if !deps.Interactive {
		return false, fmt.Errorf(
			"%w: refusing to delete %q without confirmation in a non-interactive session; use --yes to skip the prompt",
			ErrUsage, name,
		)
	}

	fmt.Fprintf(deps.Stderr, "Remove profile %q at %s? [y/N] ", name, dir)
	line, err := bufio.NewReader(deps.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
