package cli_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zipkero/ccswitch/internal/cli"
	"github.com/zipkero/ccswitch/internal/profile"
)

// storeSnapshot은 등록 파일의 실제 바이트를 그대로 읽는다. 파일이 없는 상태도 하나의 스냅샷으로
// 다루므로, 거부 경로가 등록을 고치지 않았는지와 없던 파일을 새로 만들지 않았는지가 같은 비교로
// 확인된다.
func storeSnapshot(t *testing.T, layout profile.Layout) string {
	t.Helper()
	path := layout.MetadataPath()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "<absent>"
	}
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	return string(data)
}

// listSnapshot은 list의 stdout을 그대로 돌려준다. 등록 파일이 손상된 케이스에서는 list 자신도
// 실패하므로 오류를 실패로 보지 않고, 실행 전후 결과가 같은지만 본다.
func listSnapshot(t *testing.T, layout profile.Layout) string {
	t.Helper()
	stdout, _, _ := runCommand(t, layout, "list")
	return stdout
}

// registerWithoutDir는 등록만 만들고 디렉토리는 만들지 않는다. "등록은 있는데 디렉토리 상태가
// 나쁘다"는 상태를 add로는 만들 수 없다 — add는 디렉토리를 함께 만들거나, 자리가 점유되어
// 있으면 등록도 남기지 않고 실패한다.
func registerWithoutDir(t *testing.T, layout profile.Layout, name string) {
	t.Helper()
	store, err := profile.Load(layout.MetadataPath())
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := store.Add(name); err != nil {
		t.Fatalf("Add(%q) error = %v", name, err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

// writeStoreFile은 등록 파일 자리에 임의의 내용을 그대로 쓴다. 손상된 등록 파일을 만드는 데
// 쓰는 의도적으로 비정상인 fixture다.
func writeStoreFile(t *testing.T, layout profile.Layout, content string) {
	t.Helper()
	path := layout.MetadataPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

// occupyingContent는 프로필 경로 자리에 놓는 파일의 내용이다. 거부 뒤에도 이 내용이 그대로
// 남아야 한다 — ccswitch가 그 자리를 치우지 않는다는 것이 바이트로 드러난다.
const occupyingContent = "not a profile directory"

// 실행 전 판정에 걸리는 다섯 경로가 각각 정해진 종료 코드로 끝나고, 실행 경계에 한 번도 닿지
// 않으며, 등록도 디렉토리도 만들거나 고치지 않는다.
func TestUse_RejectsWithoutTouchingProfileState(t *testing.T) {
	wantPrefix := sharedErrPrefix(t)

	cases := []struct {
		label string
		// setup은 실행 전 상태를 꾸미고 use에 줄 이름을 돌려준다.
		setup func(t *testing.T, layout profile.Layout) string
		// wantCode는 ANALYSIS §3의 코드 표에서 이 거부에 해당하는 값이다.
		wantCode int
		// wantInStderr·notInStderr는 메시지가 무엇을 가리키고 어떤 조치를 나르는지 본다.
		wantInStderr func(layout profile.Layout, name string) []string
		notInStderr  []string
		// check는 그 케이스에만 해당하는 무부작용을 본다.
		check func(t *testing.T, layout profile.Layout, name string)
	}{
		{
			label: "unregistered name",
			setup: func(t *testing.T, layout profile.Layout) string {
				return "work"
			},
			wantCode: 3,
			wantInStderr: func(layout profile.Layout, name string) []string {
				return []string{fmt.Sprintf("%q", name)}
			},
		},
		{
			label: "registered but directory deleted",
			setup: func(t *testing.T, layout profile.Layout) string {
				if _, stderr, err := runCommand(t, layout, "add", "work"); err != nil {
					t.Fatalf("add error = %v, stderr = %q", err, stderr)
				}
				if err := os.RemoveAll(layout.ProfileDir("work")); err != nil {
					t.Fatalf("RemoveAll() error = %v", err)
				}
				return "work"
			},
			wantCode: 6,
			wantInStderr: func(layout profile.Layout, name string) []string {
				// 없는 경우의 조치는 등록을 지우고 다시 만드는 절차다 (D13).
				return []string{
					layout.ProfileDir(name),
					`"ccswitch rm ` + name + `"`,
					`"ccswitch add ` + name + `"`,
				}
			},
			check: func(t *testing.T, layout profile.Layout, name string) {
				dir := layout.ProfileDir(name)
				if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("Stat(%q) error = %v, want the directory still absent", dir, err)
				}
			},
		},
		{
			label: "file at the profile path",
			setup: func(t *testing.T, layout profile.Layout) string {
				registerWithoutDir(t, layout, "work")
				if err := os.WriteFile(layout.ProfileDir("work"), []byte(occupyingContent), 0o644); err != nil {
					t.Fatalf("WriteFile() error = %v", err)
				}
				return "work"
			},
			wantCode: 6,
			wantInStderr: func(layout profile.Layout, name string) []string {
				return []string{layout.ProfileDir(name), "remove"}
			},
			// 같은 코드 6이지만 조치가 다르다 — 자리가 점유된 채로는 다시 add할 수 없으므로
			// missing 쪽 절차가 이 메시지에 섞여서는 안 된다.
			notInStderr: []string{"ccswitch add"},
			check: func(t *testing.T, layout profile.Layout, name string) {
				path := layout.ProfileDir(name)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatalf("Stat(%q) error = %v, want the file still there", path, err)
				}
				if info.IsDir() {
					t.Errorf("%q became a directory", path)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("ReadFile(%q) error = %v", path, err)
				}
				if string(data) != occupyingContent {
					t.Errorf("file content = %q, want %q", data, occupyingContent)
				}
			},
		},
		{
			label: "invalid profile name",
			setup: func(t *testing.T, layout profile.Layout) string {
				return "Work"
			},
			wantCode: 2,
		},
		{
			label: "corrupted store file",
			setup: func(t *testing.T, layout profile.Layout) string {
				writeStoreFile(t, layout, "not valid json")
				return "work"
			},
			wantCode: 1,
			wantInStderr: func(layout profile.Layout, name string) []string {
				// 손상 메시지는 이름이 아니라 어느 파일이 문제인지를 가리킨다.
				return []string{layout.MetadataPath()}
			},
			// 파일이 덮어써지지 않는다는 것은 아래 공통 스냅샷 비교로 확인된다.
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			layout := newTestLayout(t)
			name := tc.setup(t, layout)

			wantList := listSnapshot(t, layout)
			wantStore := storeSnapshot(t, layout)

			launcher := &recordingLauncher{path: fakeExecPath}
			stdout, stderr, err := runUseCLI(t, layout, launcher, nil, "use", name)

			if err == nil {
				t.Fatalf("use %q error = nil, want a rejection", name)
			}
			if got := cli.ExitCode(err); got != tc.wantCode {
				t.Errorf("ExitCode() = %d, want %d", got, tc.wantCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}

			// 거부는 다른 커맨드와 같은 접두사가 붙은 한 줄로 나온다. use만 프레임워크의 기본
			// 오류 출력을 끄고 있으므로, 새 거부 경로가 그 출력을 지나지 않으면 화면에 아무것도
			// 남지 않는다 — 그것을 여기서 잡는다.
			lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
			if len(lines) != 1 || lines[0] == "" {
				t.Fatalf("stderr = %q, want exactly one line", stderr)
			}
			if !strings.HasPrefix(lines[0], wantPrefix+" ") {
				t.Errorf("stderr line = %q, want it to start with %q", lines[0], wantPrefix)
			}
			if strings.Contains(stderr, "Usage:") {
				t.Errorf("stderr = %q, want no usage text", stderr)
			}

			if tc.wantInStderr != nil {
				for _, want := range tc.wantInStderr(layout, name) {
					if !strings.Contains(stderr, want) {
						t.Errorf("stderr = %q, want it to contain %q", stderr, want)
					}
				}
			}
			for _, unwanted := range tc.notInStderr {
				if strings.Contains(stderr, unwanted) {
					t.Errorf("stderr = %q, want it not to contain %q", stderr, unwanted)
				}
			}

			// 거부 경로는 조회에도 실행에도 닿지 않는다.
			if len(launcher.lookups) != 0 || len(launcher.specs) != 0 {
				t.Errorf("launcher was called: lookups=%q specs=%v", launcher.lookups, launcher.specs)
			}

			if got := storeSnapshot(t, layout); got != wantStore {
				t.Errorf("store file changed:\nbefore = %q\nafter  = %q", wantStore, got)
			}
			if got := listSnapshot(t, layout); got != wantList {
				t.Errorf("list output changed:\nbefore = %q\nafter  = %q", wantList, got)
			}

			if tc.check != nil {
				tc.check(t, layout, name)
			}
		})
	}
}
