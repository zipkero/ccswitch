# ccswitch

여러 Claude Code 계정을 한 머신에서 프로필 단위로 나눠 쓰는 CLI.

## 개요

Claude Code는 계정에 딸린 모든 것을 `~/.claude` 한 곳에 둔다. 자격증명, 세션 기록, 프로젝트 상태가
전부 같은 디렉토리에 쌓이기 때문에, 계정을 바꾸려면 로그아웃하고 다시 로그인하는 수밖에 없다.
개인 계정과 업무 계정을 오가는 사용자에게는 이 과정이 매번 반복된다.

ccswitch는 계정별로 독립된 설정 디렉토리를 만들어 두고, Claude Code를 실행할 때 그중 하나를
가리키게 한다. `CLAUDE_CONFIG_DIR` 환경변수를 세팅한 뒤 `claude`를 실행하는 것이 전부이며,
자격증명 파일이나 Keychain을 직접 건드리지 않는다. 계정을 어떻게 저장할지는 Claude Code가
알아서 하고, ccswitch는 어느 디렉토리를 볼지만 정해 준다.

대상 사용자는 터미널에서 Claude Code CLI를 쓰면서 계정이 둘 이상인 사람이다.

## 현재 상태

**구현 전.** 이 저장소에는 아직 코드가 없고 프로젝트 문서만 있다.

설계상 확정된 것은 다음과 같다.

- 실행 시점에 `CLAUDE_CONFIG_DIR`을 주입하는 런처 방식. 자격증명은 직접 조작하지 않는다
- Windows 우선, macOS는 이후 마일스톤
- 설정 자산(`CLAUDE.md`, `commands/`, `skills/`, `agents/`, `rules/`)은 프로필 간 공유
- 계정 정보와 세션 기록은 프로필별로 분리

명령 표면과 단계별 범위는 [ROADMAP.md](./ROADMAP.md)에 있다.

## 사용 방법

아직 없다. 설치·실행 방법은 구현이 끝난 뒤 실제로 동작하는 내용으로 이 절에 채운다.

## 운영상 주의

- `CLAUDE_CONFIG_DIR`은 Claude Code 공식 문서에 없는 환경변수다. 동작은 확인했지만
  (Claude Code 2.1.220, Windows) 비공식 동작이므로 향후 버전에서 바뀔 수 있다.
- Claude 데스크톱 앱과 IDE 확장에는 적용되지 않는다. 이들은 ccswitch를 거치지 않고 실행되며,
  앱은 자체 웹 세션으로 로그인 상태를 따로 관리한다. 자세한 근거는
  [ROADMAP.md](./ROADMAP.md)의 보류·제외 범위를 참고한다.

## 라이선스

MIT로 배포할 예정이다. `LICENSE` 파일은 아직 추가하지 않았다.

## 문서

- [ROADMAP.md](./ROADMAP.md)
