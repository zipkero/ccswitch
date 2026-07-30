package auth_test

import (
	"slices"
	"testing"

	"github.com/zipkero/ccswitch/internal/auth"
)

// LoginArgs는 하위 명령 이름 뒤에 forwarded를 순서·개수 그대로 붙인다. forwarded를 해석하지
// 않는다는 계약이 여기서 확인된다.
func TestLoginArgs(t *testing.T) {
	cases := []struct {
		name      string
		forwarded []string
		want      []string
	}{
		{name: "no forwarded arguments", forwarded: nil, want: []string{"auth", "login"}},
		{
			name:      "single flag",
			forwarded: []string{"--console"},
			want:      []string{"auth", "login", "--console"},
		},
		{
			name:      "flag with value",
			forwarded: []string{"--email", "user@example.com"},
			want:      []string{"auth", "login", "--email", "user@example.com"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := auth.LoginArgs(tc.forwarded); !slices.Equal(got, tc.want) {
				t.Errorf("LoginArgs(%q) = %q, want %q", tc.forwarded, got, tc.want)
			}
		})
	}
}

// StatusArgs는 항상 같은 세 인자를 낸다 — --json을 명시하는 것이 이 함수가 존재하는 이유다.
func TestStatusArgs(t *testing.T) {
	want := []string{"auth", "status", "--json"}
	if got := auth.StatusArgs(); !slices.Equal(got, want) {
		t.Errorf("StatusArgs() = %q, want %q", got, want)
	}
}

// ReadStatus의 판정은 종료 코드가 아니라 stdout이 loggedIn을 담은 JSON 객체로 읽히는지에
// 달려 있다(D4) — 실기에서 관찰된 대로, 로그인되지 않은 상태도 종료 코드 1로 보고된다.
func TestReadStatus(t *testing.T) {
	cases := []struct {
		name     string
		stdout   string
		stderr   string
		exitCode int
		wantErr  bool
		want     auth.Status
	}{
		{
			name:     "logged out reported as exit code 1 is still an answer",
			stdout:   `{"loggedIn": false, "authMethod": "none", "apiProvider": "firstParty"}`,
			exitCode: 1,
			want:     auth.Status{LoggedIn: false, AuthMethod: "none", APIProvider: "firstParty"},
		},
		{
			name:     "logged in with full fields",
			stdout:   `{"loggedIn": true, "authMethod": "claudeai", "apiProvider": "firstParty", "email": "work@example.com", "orgId": "org_1", "orgName": "Acme Inc", "subscriptionType": "max"}`,
			exitCode: 0,
			want: auth.Status{
				LoggedIn:         true,
				AuthMethod:       "claudeai",
				APIProvider:      "firstParty",
				Email:            "work@example.com",
				OrgID:            "org_1",
				OrgName:          "Acme Inc",
				SubscriptionType: "max",
			},
		},
		{
			name:     "JSON object without loggedIn field is a failure",
			stdout:   `{"authMethod": "none", "apiProvider": "firstParty"}`,
			exitCode: 0,
			wantErr:  true,
		},
		{
			name:     "non-JSON output is a failure",
			stdout:   "command not found",
			stderr:   "claude: command not found",
			exitCode: 127,
			wantErr:  true,
		},
		{
			name:     "JSON array is not an object and is a failure",
			stdout:   `[{"loggedIn": true}]`,
			exitCode: 0,
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := auth.ReadStatus(tc.stdout, tc.stderr, tc.exitCode)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ReadStatus() error = nil, want a failure")
				}
				return
			}
			if err != nil {
				t.Fatalf("ReadStatus() error = %v", err)
			}
			if got != tc.want {
				t.Errorf("ReadStatus() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
