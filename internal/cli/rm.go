package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/auth"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newRmCommand(deps Deps) *cobra.Command {
	var yes, skipLogout bool
	cmd := &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a profile's directory and registration",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRm(deps, args[0], yes, skipLogout)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	// 되돌릴 수 없는 결과를 넓히는 플래그라 짧은 형태는 두지 않는다(profile-auth D11) — 전부
	// 타이핑하게 둔다.
	cmd.Flags().BoolVar(&skipLogout, "skip-logout", false, "skip the Claude Code logout delegation before deleting; credentials may remain")
	return cmd
}

func runRm(deps Deps, name string, yes, skipLogout bool) error {
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

	// --skip-logout이면 PATH 조회도 위임도 걸지 않는다(profile-auth D11) — 걸지 않을 위임을
	// 두고 조회할 이유가 없다. 경고는 정리를 건너뛴다는 결정이 확정되는 이 자리에서, 확인
	// 프롬프트보다 먼저 낸다 — 사용자가 승인하는 대상에 "자격증명은 남는다"가 들어가야 한다.
	var path string
	if skipLogout {
		fmt.Fprintf(deps.Stderr, "Warning: skipping the Claude Code logout for %q; credentials may remain\n", name)
	} else {
		// PATH 조회는 확인 프롬프트보다 앞이다(profile-auth D9) — 정리를 실행할 수 없다는 사실은
		// 승인을 받기 전에 알려야 한다. 승인 뒤에야 조회가 실패하면 "정리한 다음 지운다"는 승인의
		// 뜻이 깨진다.
		var err error
		path, err = lookupClaudeExecutable(deps)
		if err != nil {
			return err
		}
	}

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

	if !skipLogout {
		// 정리 위임은 디렉토리 삭제보다 앞이며, 위임 앞에 디렉토리 상태를 미리 판정하지 않는다 —
		// claude auth logout은 설정 디렉토리가 없어도 종료 코드 0으로 끝나고, macOS에서
		// 자격증명은 디렉토리 밖 Keychain에 있으므로 디렉토리가 이미 사라진 프로필에도 위임을
		// 걸어야 잔여물이 없어진다(profile-auth D10). 로그인 여부를 미리 조회하지도 않는다 —
		// 그 조회 자체가 부프로세스를 하나 더 띄우고 없는 디렉토리를 만들어 버린다(profile-auth
		// D10).
		captured, captureErr := deps.Launcher.Capture(launch.Spec{
			Path: path,
			Args: auth.LogoutArgs(),
			Env:  deps.Platform.Environ(deps.BaseEnv, dir),
		})
		if captureErr == nil {
			captureErr = auth.ReadLogout(captured.Stdout, captured.Stderr, captured.ExitCode)
		}
		if captureErr != nil {
			// 위임이 실패하면 디렉토리도 등록도 건드리지 않는다(SPEC §5.8). 실패 원인은 위임한
			// 명령이 답을 주지 못한 것이지 프로필의 로그인 여부 문제가 아니므로, logout과 같은
			// 기준으로 상태를 바꾸는 명령을 조치로 권하지 않는다.
			return fmt.Errorf(
				"%w: %v; check that %q is a working Claude Code installation, or reinstall it",
				ErrClaudeAuthFailed, captureErr, path,
			)
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
