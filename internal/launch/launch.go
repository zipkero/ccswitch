// Package launch는 Claude Code 프로세스에 무엇을 넘길지 계산하는 일과, 실제로 프로세스를
// 띄워 기다리는 일만 소유한다. 프로필 등록·종료 코드 체계·인자 파싱을 알지 못하며
// internal/cli를 참조하지 않는다.
//
// 계산은 순수 부분(Platform.Environ)에, 프로세스 세계와의 접촉은 Launcher 구현에 모여 있다.
// 테스트는 Launcher를 기록용 대역으로 갈아끼워 프로세스를 띄우지 않고 무엇을 넘겼는지 본다.
package launch

// Spec은 자식 프로세스에 넘길 것 전부다. 작업 디렉토리는 담지 않는다 — 자식은 부모의 현재
// 디렉토리를 그대로 물려받아야 Claude Code가 같은 프로젝트를 본다.
type Spec struct {
	// Path는 PATH 조회로 확정된 실행 파일 경로다.
	Path string
	// Args는 자식에게 넘길 인자 전체다. 실행 파일 이름은 포함하지 않는다. use처럼 사용자가
	// "--" 뒤에 준 값만 담을 수도 있고, login처럼 ccswitch가 정한 하위 명령(auth login 등)이
	// 앞에 붙은 전체 목록일 수도 있다.
	Args []string
	// Env는 자식에게 줄 환경 전체다. 상속에 맡기지 않고 항상 채운다 — 상속으로 두면
	// Platform.Environ이 지운 항목이 그대로 살아난다.
	Env []string
}

// Captured는 출력을 받아 온 실행의 결과다. 0이 아닌 종료 코드는 error가 아니다 — Run과 같은
// 규칙이며, claude auth status가 로그인되지 않은 상태를 1로 보고하기 때문에 특히 그렇다.
type Captured struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Launcher는 프로세스 세계와의 유일한 접점이다.
type Launcher interface {
	// Lookup은 PATH에서 실행 파일을 찾아 그 경로를 돌려준다. 찾지 못하면
	// exec.ErrNotFound를 감싼 error를 돌려주므로 호출자가 errors.Is로 가릴 수 있다.
	Lookup(name string) (string, error)
	// Run은 자식이 끝날 때까지 기다린다. 자식이 끝났으면 그 종료 코드와 nil을 돌려주며,
	// 0이 아닌 종료 코드도 error가 아니다 — error는 띄우지 못했거나 기다릴 수 없었던
	// 경우만을 가리킨다.
	Run(spec Spec) (int, error)
	// Capture는 자식을 띄워 끝날 때까지 기다리고 표준 출력·표준 오류를 값으로 받아 온다.
	// 부모 콘솔을 자식에게 넘기지 않고, 자식의 표준 입력은 비어 있다 — 무언가를 묻는 자식이
	// 입력을 기다리며 멈추지 않게 한다. 인터럽트 무시를 걸지 않으므로 Ctrl+C가 둘을 함께
	// 끝낸다.
	Capture(spec Spec) (Captured, error)
}
