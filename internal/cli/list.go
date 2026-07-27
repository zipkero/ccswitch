package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/zipkero/ccswitch/internal/profile"
)

func newListCommand(deps Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered profiles",
		Args:  noArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(deps)
		},
	}
}

func runList(deps Deps) error {
	store, err := profile.Load(deps.Layout.MetadataPath())
	if err != nil {
		return err
	}

	// default는 등록에 저장되지 않고 항상 맨 위에 합성된다 (ANALYSIS D3). 이어지는
	// 등록 이름은 Store.Names()가 이미 사전순으로 돌려준다.
	type row struct {
		name string
		dir  string
	}
	rows := make([]row, 0, 1+len(store.Names()))
	rows = append(rows, row{name: profile.DefaultName, dir: deps.Layout.DefaultDir()})
	for _, name := range store.Names() {
		rows = append(rows, row{name: name, dir: deps.Layout.ProfileDir(name)})
	}

	w := tabwriter.NewWriter(deps.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDIR\tSTATUS")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\n", r.name, r.dir, profile.Inspect(r.dir))
	}
	return w.Flush()
}
