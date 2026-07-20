package pnpm

import (
	"os"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/util"
	"gopkg.in/yaml.v3"
)

type workspaceYaml struct {
	Packages []string `yaml:"packages"`
}

// ActionWorkspaces completes workspaces
//
//	packages/a
//	packages/b
func ActionWorkspaces() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if workspaceFile, err := util.FindReverse(c.Dir, "pnpm-workspace.yaml"); err == nil {
			return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
				content, err := os.ReadFile(workspaceFile)
				if err != nil {
					return carapace.ActionMessage(err.Error())
				}

				var ws workspaceYaml
				if err := yaml.Unmarshal(content, &ws); err != nil {
					return carapace.ActionMessage(err.Error())
				}

				return carapace.ActionValues(ws.Packages...)
			})
		}

		if pj, err := loadPackageJson(c); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			return carapace.ActionValues(pj.Workspaces...)
		}
	})
}
