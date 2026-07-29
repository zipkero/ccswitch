# ccswitch

여러 Claude Code 계정을 한 머신에서 프로필 단위로 나눠 쓰는 CLI.

> **상태: `add`·`list`·`use`·`rm`까지 동작한다.** `ccswitch use work`로 계정을 바꿔 Claude Code를
> 띄울 수 있다. 로그인은 아직 그 세션 안에서 직접 하고, `list`는 이름·경로·상태만 보여준다.
> 나머지는 [ROADMAP.md](./ROADMAP.md)에 있다.

## 왜 필요한가

Claude Code는 계정에 딸린 모든 것을 `~/.claude` 한 곳에 둔다. 자격증명도, 세션 기록도,
프로젝트 상태도 같은 디렉토리에 쌓인다. 그래서 개인 계정과 업무 계정을 오가려면 매번
로그아웃하고 다시 로그인하는 수밖에 없다.

ccswitch는 계정마다 설정 디렉토리를 따로 두고, Claude Code를 실행할 때 어느 쪽을 볼지만
지정한다.

## 써보기

프로필을 만든다. 이름은 소문자·숫자로 시작해 소문자·숫자·`-`·`_`로 32자까지 쓸 수 있으며,
`default`는 예약되어 있다.

```bash
ccswitch add work
```

```
Created profile "work" at C:\Users\you\.claude-work
```

등록된 프로필을 확인한다. `default`는 등록에 저장되지 않고 Claude Code의 기본 설정 디렉토리를
가리키는 항목으로 항상 맨 위에 나온다. `STATUS`는 조회 시점에 그 경로를 실제로 조사한 결과이며,
디렉토리가 없으면 `missing`, 디렉토리가 아닌 것이 놓여 있으면 `unusable`이 된다.

```bash
ccswitch list
```

```
NAME     DIR                        STATUS
default  C:\Users\you\.claude       ok
work     C:\Users\you\.claude-work  ok
```

그 프로필로 Claude Code를 띄운다. 새 창을 열지 않고 같은 창에서 그대로 실행되며, 성공하면
ccswitch 자신은 한 줄도 출력하지 않는다.

```bash
ccswitch use work
```

로그인은 ccswitch가 다루지 않는다. 새로 만든 프로필로 처음 띄운 세션은 아직 어느 계정에도
로그인되어 있지 않으니, 그 세션 안에서 Claude Code의 로그인 절차를 그대로 밟는다. 자격증명은
그 프로필 디렉토리에 저장되므로, 그 뒤로는 `ccswitch use work`가 곧 그 계정으로 실행하는 것이
된다. 계정을 하나 더 쓰려면 다른 이름으로 프로필을 만들어 같은 절차를 반복한다.

이름을 생략하거나 `default`를 주면 기존 `~/.claude`를 쓰는 세션이 열린다 — `claude`를 직접
실행한 것과 같다.

```bash
ccswitch use
ccswitch use default
```

`--` 뒤에 준 값은 ccswitch가 해석하지 않고 `claude`에 그대로 넘긴다. 세션이 끝나면 Claude Code의
종료 코드가 ccswitch의 종료 코드로 그대로 나온다.

```bash
ccswitch use work -- --model opus
```

프로필을 지우면 디렉토리와 등록이 함께 사라진다. 확인 프롬프트를 건너뛰려면 `-y`를 준다.

```bash
ccswitch rm work
```

계획하고 있는 명령은 [ROADMAP.md](./ROADMAP.md#포함-범위)의 명령 표면 표에 있다.

## 무엇이 나뉘고 무엇이 공유되는가

| 프로필마다 따로 (지금) | 프로필 사이에 공유 (계획) |
|---|---|
| 자격증명 | `CLAUDE.md` |
| 세션·대화 기록 | `commands/` `skills/` `agents/` `rules/` |
| 프로젝트 상태, MCP 설정 | `settings.json`, 설정 디렉토리 루트의 외부 도구 설정 파일 |

왼쪽은 지금 그렇게 동작한다. 프로필마다 설정 디렉토리가 따로이므로 그 안에 쌓이는 것은 전부
갈린다.

오른쪽은 아직이다. 새 프로필은 빈 디렉토리로 시작해서, default의 메모리·스킬·커맨드·에이전트가
따라오지 않는다. 계정은 갈리지만 손에 익은 작업 환경은 따라오게 하는 것이 목표이고, 그것은
[M3](./ROADMAP.md#m3-설정-자산을-프로필-간-공유한다)에서 채운다.

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
- **구독 계정 로그인은 브라우저를 거친다.** ccswitch는 자격증명을 만들지 않으므로 프로필마다
  한 번은 Claude Code의 로그인 절차를 밟아야 한다. 이때 브라우저에 다른 계정이 이미 붙어
  있으면 그 계정으로 인증이 통과해, 프로필은 둘인데 계정은 하나가 될 수 있다. 두 번째 계정은
  로그아웃한 상태나 시크릿 창에서 받는다. 프로필당 최초 1회이며 그 뒤의 `use`는 브라우저를
  거치지 않는다.
- `CLAUDE_CONFIG_DIR`은 Claude Code 공식 문서에 없는 환경변수다. Claude Code 2.1.220의 Windows와
  macOS에서 동작을 확인했지만 비공식이라 향후 버전에서 바뀔 수 있다.
- Windows 우선이고 macOS는 이후 마일스톤이다.

## 문서

- [ROADMAP.md](./ROADMAP.md) — 최종 결과물, 완료 기준, 마일스톤, 보류 범위

## 라이선스

MIT 예정. `LICENSE` 파일은 아직 없다.
