# ccswitch

여러 Claude Code 계정을 한 머신에서 프로필 단위로 나눠 쓰는 CLI.

> **상태: 프로필 생성·조회·삭제(`add`·`list`·`rm`)까지 동작.** 계정을 바꿔 실행하는 `use`가 아직 없어
> 설치해도 계정 전환에는 쓸 수 없습니다.
> 아래 사용 예시는 계획된 인터페이스이며, `list`의 계정 정보 표시를 포함해 상당 부분이 아직 검증된
> 동작이 아닙니다.

## 왜 필요한가

Claude Code는 계정에 딸린 모든 것을 `~/.claude` 한 곳에 둔다. 자격증명도, 세션 기록도,
프로젝트 상태도 같은 디렉토리에 쌓인다. 그래서 개인 계정과 업무 계정을 오가려면 매번
로그아웃하고 다시 로그인하는 수밖에 없다.

ccswitch는 계정마다 설정 디렉토리를 따로 두고, Claude Code를 실행할 때 어느 쪽을 볼지만
지정한다.

## 사용 예시 (계획)

```bash
ccswitch add work        # 프로필 생성
ccswitch login work      # 그 프로필로 로그인
ccswitch use work        # work 계정으로 Claude Code 실행

ccswitch use work -- --model opus     # -- 뒤는 claude 로 그대로 전달
claude                                # 프로필을 안 거치면 그대로 default 계정
```

```bash
ccswitch list
```

```
NAME     ACCOUNT             PLAN
default  me@personal.com     pro
work     me@company.com      team
```

전체 명령 목록은 [ROADMAP.md](./ROADMAP.md#포함-범위)에 있다.

## 무엇이 나뉘고 무엇이 공유되는가

| 프로필마다 따로 | 프로필 사이에 공유 |
|---|---|
| 자격증명 | `CLAUDE.md` |
| 세션·대화 기록 | `commands/` `skills/` `agents/` `rules/` |
| 프로젝트 상태, MCP 설정 | `settings.json`, 설정 디렉토리 루트의 외부 도구 설정 파일 (명시적 동기) |

계정은 갈리지만 손에 익은 작업 환경은 따라온다. 새 프로필이 빈손으로 시작하지 않는다.

명시적 동기 대상은 프로필을 만들 때 복사해 두고, 그 뒤로는 프로필마다 다르게 고쳐 쓸 수 있다.
`sync`를 부를 때만 원본 기준으로 되돌아간다.

## 동작 방식

실행 직전에 `CLAUDE_CONFIG_DIR`을 프로필 디렉토리로 세팅하고 `claude`를 띄운다. 그게 전부다.
default 프로필만 예외로, 변수를 세팅하는 대신 환경에서 지운다 — 기본 경로를 값으로 넣으면 Claude
Code가 평소와 다른 자격증명 항목과 다른 설정 파일 위치를 보게 된다.

자격증명 파일이나 Keychain은 직접 건드리지 않는다. 계정을 어디에 어떻게 저장할지는 Claude Code가
하던 대로 하고, ccswitch는 어느 디렉토리를 볼지만 정해 준다. 그래서 플랫폼마다 저장 방식이 달라도
같은 방식으로 동작하고, 잘못돼도 되돌릴 것이 없다. macOS에서는 Claude Code가 Keychain 항목 이름에
설정 디렉토리 경로의 해시를 붙여, 프로필마다 다른 항목을 쓴다(2.1.220 확인).

## 알아둘 것

- **터미널 CLI 전용이다.** Claude 데스크톱 앱과 IDE 확장에는 적용되지 않는다.
  이들은 ccswitch를 거치지 않고 실행되며, 앱은 로그인 상태를 따로 관리한다.
- `CLAUDE_CONFIG_DIR`은 Claude Code 공식 문서에 없는 환경변수다. Claude Code 2.1.220의 Windows와
  macOS에서 동작을 확인했지만 비공식이라 향후 버전에서 바뀔 수 있다.
- Windows 우선이고 macOS는 이후 마일스톤이다.

## 문서

- [ROADMAP.md](./ROADMAP.md) — 최종 결과물, 완료 기준, 마일스톤, 보류 범위

## 라이선스

MIT 예정. `LICENSE` 파일은 아직 없다.
