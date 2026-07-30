package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/auth"
	"github.com/zipkero/ccswitch/internal/launch"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newListCommand(deps Deps) *cobra.Command {
	var accounts bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered profiles",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(deps, accounts)
		},
	}
	// 짧은 형태를 두지 않는다 — 이 옵션은 프로필마다 부프로세스를 띄우는 느린 조회를 일부러
	// 요청하는 자리라, -y처럼 반복 타이핑을 줄여야 할 이유가 없다(ANALYSIS D6).
	cmd.Flags().BoolVar(&accounts, "accounts", false, "show each profile's account and plan via claude auth status")
	return cmd
}

// listRow는 한 행에 낼 정보를 담는다. status는 accounts 옵션 여부와 무관하게 항상 한 번만
// 계산해 STATUS 열 출력과 조회 대상 판정(ANALYSIS D7)에 함께 쓴다.
type listRow struct {
	name    string
	dir     string
	status  profile.Status
	account string
	plan    string
}

func runList(deps Deps, accounts bool) error {
	store, err := profile.Load(deps.Layout.MetadataPath())
	if err != nil {
		return err
	}

	// default는 등록에 저장되지 않고 항상 맨 위에 합성된다 (ANALYSIS D3). 이어지는
	// 등록 이름은 Store.Names()가 이미 사전순으로 돌려준다.
	names := append([]string{profile.DefaultName}, store.Names()...)
	rows := make([]listRow, len(names))
	for i, name := range names {
		dir := deps.Layout.ProfileDir(name)
		if name == profile.DefaultName {
			dir = deps.Layout.DefaultDir()
		}
		rows[i] = listRow{name: name, dir: dir, status: profile.Inspect(dir)}
	}

	failed := false
	if accounts {
		// PATH 조회는 옵션이 있을 때만, 프로필마다가 아니라 한 번만 한다(ANALYSIS D6). 실패하면
		// 표를 내지 않고 코드 7이다 — use·login과 같은 메시지를 그대로 재사용한다.
		path, err := lookupClaudeExecutable(deps)
		if err != nil {
			return err
		}
		for i := range rows {
			if rows[i].status != profile.StatusOK {
				// 상태가 ok가 아닌 행은 조회하지 않는다 — claude auth status가 없는 설정
				// 디렉토리를 만들어 버리기 때문이다(ANALYSIS D7). 조회하지 않은 것은 실패가
				// 아니므로 종료 코드에 영향을 주지 않는다.
				rows[i].account, rows[i].plan = "-", "-"
				continue
			}
			if err := queryAccount(deps, path, &rows[i]); err != nil {
				failed = true
				// 이 분기에 오는 원인은 claude 프로세스를 띄우지 못했거나 출력이 기대한
				// JSON이 아닌 경우뿐이다 — 로그인되지 않은 상태는 정상 응답이라 queryAccount의
				// LoggedIn 분기로 이미 빠진다. 즉 원인은 항상 이 실행 파일 쪽 문제이지 프로필의
				// 로그인 여부가 아니므로, 상태를 바꾸는 login을 권하지 않는다.
				fmt.Fprintf(deps.Stderr,
					"%q: could not read account status (%v); check that %q is a working Claude Code installation, or reinstall it\n",
					rows[i].name, err, path)
			}
		}
	}

	w := tabwriter.NewWriter(deps.Stdout, 0, 4, 2, ' ', 0)
	if accounts {
		fmt.Fprintln(w, "NAME\tDIR\tSTATUS\tACCOUNT\tPLAN")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", r.name, r.dir, r.status, r.account, r.plan)
		}
	} else {
		fmt.Fprintln(w, "NAME\tDIR\tSTATUS")
		for _, r := range rows {
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.name, r.dir, r.status)
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	// 조회 실패가 하나라도 있으면 표를 이미 낸 뒤에 실패로 끝난다 — 프로필 하나의 조회가
	// 실패했다고 나머지 정보를 감출 이유가 없다(SPEC §5.11).
	if failed {
		return ErrClaudeAuthFailed
	}
	return nil
}

// queryAccount는 profile.StatusOK인 행 하나를 claude auth status --json으로 조회해
// row.account·row.plan을 채운다. default 행은 설정 디렉토리를 주입하지 않는다 — use·login과
// 같은 대상 해석이다.
func queryAccount(deps Deps, path string, row *listRow) error {
	configDir := row.dir
	if row.name == profile.DefaultName {
		configDir = ""
	}
	captured, err := deps.Launcher.Capture(launch.Spec{
		Path: path,
		Args: auth.StatusArgs(),
		Env:  deps.Platform.Environ(deps.BaseEnv, configDir),
	})
	if err != nil {
		row.account, row.plan = "unknown", "-"
		return err
	}

	status, err := auth.ReadStatus(captured.Stdout, captured.Stderr, captured.ExitCode)
	if err != nil {
		row.account, row.plan = "unknown", "-"
		return err
	}

	if !status.LoggedIn {
		row.account, row.plan = "logged-out", "-"
		return nil
	}
	account := status.Email
	if account == "" {
		account = status.AuthMethod
	}
	plan := status.SubscriptionType
	if plan == "" {
		plan = "-"
	}
	row.account, row.plan = account, plan
	return nil
}
