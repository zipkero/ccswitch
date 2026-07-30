// Package auth는 claude auth 하위 명령 인자를 만들고 그 결과를 판정하는 순수 경계다. 하위
// 명령 이름과 옵션, 그리고 claude auth status의 출력을 어떻게 읽는지를 아는 유일한 자리이며,
// 프로세스를 직접 띄우지 않는다. internal/launch·internal/cli를 참조하지 않는다.
package auth

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LoginArgs는 claude auth login에 넘길 인자를 만든다. forwarded는 사용자가 "--" 뒤에 준
// 값이며 해석하지 않고 그대로 뒤에 붙인다.
func LoginArgs(forwarded []string) []string {
	args := make([]string, 0, len(forwarded)+2)
	args = append(args, "auth", "login")
	return append(args, forwarded...)
}

// StatusArgs는 claude auth status에 넘길 인자를 만든다. --json은 지금 기본값이지만 명시해
// 둔다 — 기본값이 바뀌면 ReadStatus의 파싱이 조용히 깨지기 때문이다.
func StatusArgs() []string {
	return []string{"auth", "status", "--json"}
}

// LogoutArgs는 claude auth logout에 넘길 인자를 만든다. 이 하위 명령은 옵션이 없으므로(§근거)
// 추가 인자를 받지 않는다.
func LogoutArgs() []string {
	return []string{"auth", "logout"}
}

// Status는 claude auth status가 보고한 인증 상태다. 로그인되지 않은 상태에서는 계정 식별
// 필드가 비어 있다 — Claude Code가 그 필드를 아예 내지 않는다.
type Status struct {
	LoggedIn         bool
	AuthMethod       string
	APIProvider      string
	Email            string
	OrgID            string
	OrgName          string
	SubscriptionType string
}

// statusJSON은 claude auth status --json이 내는 필드 이름을 그대로 옮긴다. LoggedIn을
// 포인터로 두는 것은 그 필드가 아예 없는 JSON(다른 도구의 출력, 손상된 출력 등)과 그 필드가
// false인 JSON을 구별하기 위해서다 — 값 자체가 아니라 필드의 유무가 판정을 가른다.
type statusJSON struct {
	LoggedIn         *bool  `json:"loggedIn"`
	AuthMethod       string `json:"authMethod"`
	APIProvider      string `json:"apiProvider"`
	Email            string `json:"email"`
	OrgID            string `json:"orgId"`
	OrgName          string `json:"orgName"`
	SubscriptionType string `json:"subscriptionType"`
}

// ReadStatus는 끝난 claude auth status의 결과를 읽는다. stdout이 loggedIn을 담은 JSON 객체로
// 읽히면 종료 코드가 무엇이든 그것이 답이다 — Claude Code는 로그인되지 않은 상태를 종료 코드
// 1로 보고한다. 그렇게 읽히지 않으면 조회가 실패한 것으로 보고, 캡처한 출력과 종료 코드를
// 담은 error를 돌려준다.
func ReadStatus(stdout, stderr string, exitCode int) (Status, error) {
	var raw statusJSON
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil || raw.LoggedIn == nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = strings.TrimSpace(stdout)
		}
		return Status{}, fmt.Errorf(
			`claude auth status exited %d without a JSON object containing "loggedIn": %s`,
			exitCode, detail,
		)
	}
	return Status{
		LoggedIn:         *raw.LoggedIn,
		AuthMethod:       raw.AuthMethod,
		APIProvider:      raw.APIProvider,
		Email:            raw.Email,
		OrgID:            raw.OrgID,
		OrgName:          raw.OrgName,
		SubscriptionType: raw.SubscriptionType,
	}, nil
}

// ReadLogout은 끝난 claude auth logout의 결과를 읽는다. 0이면 성공이고, 0이 아니면 캡처한
// 출력을 실패 이유로 담은 error를 돌려준다.
func ReadLogout(stdout, stderr string, exitCode int) error {
	if exitCode == 0 {
		return nil
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" {
		detail = strings.TrimSpace(stdout)
	}
	return fmt.Errorf("claude auth logout exited %d: %s", exitCode, detail)
}
