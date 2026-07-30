# Context

## 현재 목표

profile-auth feature를 완료해 계정을 프로필에 붙이고 떼는 일을 ccswitch 안에서 끝낸다 —
`login`·`logout`·계정 조회·삭제 시 자격증명 정리를 모두 Claude Code에 위임한다.

## 현재 상태

SPEC·ANALYSIS·IMPLEMENT 체크리스트 작성이 끝났고, 코드 Task 5개(task-001~005)가 구현·검증까지
완료됐다. `login`·`logout`이 커맨드 트리에 올라왔고 `go build`·`go vet`·`gofmt`·
`go test -count=1 ./...`가 통과한다.

`/implement-loop`는 task-006에서 정지했다 — 그 Task의 검증이 브라우저 OAuth와 서로 다른 실계정
두 개를 요구해 자동으로 진행할 수 없다(자동 진행 제외). 남은 두 Task(task-006 실기 관찰,
task-007 README)는 둘 다 사용자 관찰이 완료 근거다.

변경은 커밋하지 않은 상태다.

## 현재 작업 문서

- [features/20260730-001-profile-auth/implement.md](./features/20260730-001-profile-auth/implement.md)
  — 다음 대상은 task-006, 그 뒤 task-007

## 확정된 결정

사용자가 확정한 것 (analysis.md §5와 spec.md에 반영 완료):

- 계정 조회 옵션은 `--accounts`, 열은 `ACCOUNT`·`PLAN`. 표기는 `logged-out`(미로그인) /
  `unknown`(조회 실패) / `-`(조회 안 함). 짧은 형태를 두지 않고, 공백이 들어가는 `orgName`은
  열 값으로 쓰지 않는다 (D6).
- 정리를 건너뛰는 수단은 `--skip-logout` 하나이고 그 경로에 추가 확인을 겹치지 않는다. 경고는
  프롬프트보다 앞에서 낸다 (D11).
- 이번 feature가 더하는 실패 셋(logout 실패 / rm 정리 실패 / 계정 조회 실패)을 종료 코드 8
  하나로 묶는다. 9 이후는 이후 feature에 비워 둔다 (D12).
- 루트 README는 상태 문구와 사용 절차만 고치고 설치 안내는 넣지 않는다 (D14).
- spec.md §5에 12번(README가 실제 동작과 일치)을 추가했다 — D14가 확정한 작업에 걸 완료 조건이
  없어 Task를 만들 수 없었고, 선행 profile-launcher에는 같은 자리에 §5.12가 있었다.

코드베이스에서 확인한 것:

- `claude auth status`는 로그인되지 않은 상태를 **종료 코드 1**로 보고한다. 그래서 판정은 종료
  코드를 쓰지 않고 stdout이 `loggedIn`을 담은 JSON 객체로 읽히는지로 가른다 (D4).
- `claude auth status`·`logout`은 없는 설정 디렉토리를 **만든다**. 그래서 `list --accounts`는
  `Inspect`가 `ok`인 행만 조회한다 (D7).
- `claude auth logout`은 미로그인·없는 디렉토리에서도 종료 코드 0으로 끝난다. SPEC §5.10은 이
  멱등성으로 성립하며 별도 사전 판정을 두지 않는다 (D10).
- `backups/`는 실행 횟수로 쌓이지 않는다(9회 실행에 1개). 반복 조회가 파일을 축적하지 않는다.

세션 운영 지침으로 사용자가 확정한 것:

- subagent가 근거를 모을 때 **실제 `claude` 실행 파일을 부를 수 있는 경로를 만들지 않는다.**
  실제 `claude auth login`이 떠 브라우저 OAuth 탭이 열린 사고가 있었다.

## 미확정 판단

아래 셋은 모두 검증 과정에서 나온 미논의 후보이며, 사용자와 논의한 적이 없다.

- 테스트에서 ccswitch를 격리하려면 `Home`과 `ConfigRoot`를 **둘 다** 임시 디렉토리로 채워야
  한다 — `HOME`만 바꾸면 `Layout.ConfigRoot`가 `os.UserConfigDir()`에서 오므로 실제
  `%AppData%\ccswitch\profiles.json`이 오염된다. 검증 중 실제로 오염됐다가 되돌린 사고가 있었다.
  이 함정을 테스트 헬퍼 주석이나 `CLAUDE.md`에 남길지가 미정이다. 근거는
  [internal/profile/paths.go](./internal/profile/paths.go)의 `NewLayout`·`MetadataPath`.
- `list --accounts`의 조회 실패 조치 문구가 테스트로 고정되지 않았다. 그 문구는 한 번 reject된
  적이 있고(상태를 바꾸는 `ccswitch login`을 권했다) 현재는 주석만이 되돌림을 막는다.
  `internal/cli/use_lookup_test.go`가 쓰는 부분 문자열 단정 방식을 더할지가 미정이다.
- `--skip-logout`의 축약형 회귀 테스트가 `-s` 한 글자만 막는다. 다른 문자를 shorthand로 붙이는
  회귀는 잡히지 않는다. 더 넓게 막을지가 미정이다.

## 다음 작업

- 작업: task-006 실기 관찰. `go build -o ccswitch.exe .`로 빌드한 뒤 서로 다른 두 계정으로
  프로필 두 개를 만들어 (1) 한쪽 `login` 뒤 `list --accounts`에서 그 행만 계정이 채워지고
  `default`·다른 프로필 행은 그대로인지, (2) 다른 쪽도 다른 계정으로 같은 절차를 밟아 두 행이
  서로 다른 계정을 보여주는지, (3) 한쪽 `logout` 뒤 그 행만 `logged-out`이 되는지를 확인한다.
- 완료 기준: 세 관찰이 모두 성립해 SPEC §5.1·§5.2·§5.3이 닫히고 implement.md task-006이
  `[x]`가 된다. 어긋나는 관찰이 나오면 그 동작을 소유한 앞선 Task(task-001 또는 task-003)로
  되돌린다.

## 먼저 읽을 문서

- [features/20260730-001-profile-auth/implement.md](./features/20260730-001-profile-auth/implement.md)
  — task-006의 검증 조건, 이어서 task-007
- [features/20260730-001-profile-auth/spec.md](./features/20260730-001-profile-auth/spec.md)
  — §5.1·§5.2·§5.3, 그리고 task-007이 닫는 §5.12
- [features/20260730-001-profile-auth/analysis.md](./features/20260730-001-profile-auth/analysis.md)
  — task-007이 따르는 §4와 D14

## 문서 반영 필요

없음.
