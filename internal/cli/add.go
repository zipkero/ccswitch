package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newAddCommand(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "add <name>",
		Short: "Create and register a new profile",
		Args:  exactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdd(deps, args[0])
		},
	}
}

func runAdd(deps Deps, name string) error {
	// 이름 검증은 파일시스템·등록 파일 접근보다 먼저 끝나야 한다 — 형식 위반이나 예약
	// 이름은 디스크에 전혀 닿지 않고 거부된다.
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	store, err := profile.Load(deps.Layout.MetadataPath())
	if err != nil {
		return err
	}

	// 등록 여부를 대상 경로 조사보다 먼저 확인해, 등록 충돌이 파일시스템에 닿기 전에
	// 끝나게 한다. 여기서 Store.Add는 메모리 상의 판정일 뿐이고 실제 저장(Save)은 경로
	// 조사를 통과한 뒤에 하므로, 뒤이은 EnsureProfileDir이 실패해도 파일은 그대로 남는다.
	if err := store.Add(name); err != nil {
		return err
	}

	dir := deps.Layout.ProfileDir(name)
	if err := profile.EnsureProfileDir(dir); err != nil {
		return err
	}

	if err := store.Save(); err != nil {
		return err
	}

	fmt.Fprintf(deps.Stderr, "Created profile %q at %s\n", name, dir)
	return nil
}
