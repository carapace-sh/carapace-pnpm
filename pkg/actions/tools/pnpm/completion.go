package pnpm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/style"
)

// pnpmListEntry is one element of `pnpm list --json -r --depth -1` output.
type pnpmListEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
	Private bool   `json:"private"`
}

// ActionWorkspacePackages completes workspace package names by running
// `pnpm list --json -r --depth -1`. Each candidate is the package's name with
// its version as the description. Falls back to an empty action if pnpm is
// unavailable or not in a workspace.
//
//	@scope/widget (1.0.0)
//	shared        (2.1.0)
//	web-app       (0.1.0)
func ActionWorkspacePackages() carapace.Action {
	return carapace.ActionExecCommandE("pnpm", "list", "--json", "-r", "--depth", "-1")(func(output []byte, err error) carapace.Action {
		var entries []pnpmListEntry
		if len(output) == 0 || json.Unmarshal(output, &entries) != nil {
			return carapace.ActionValues()
		}
		vals := make([]string, 0, len(entries)*2)
		for _, e := range entries {
			if e.Name == "" {
				continue
			}
			desc := e.Version
			if e.Private {
				if desc != "" {
					desc += " "
				}
				desc += "(private)"
			}
			vals = append(vals, e.Name, desc)
		}
		return carapace.ActionValuesDescribed(vals...).Tag("workspace package").UidF(Uid("workspace-package"))
	}).Cache(0)
}

// ActionWorkspacePaths completes workspace package paths (relative to the
// current directory) by running `pnpm list --json -r --depth -1`. Each
// candidate is a `./`-prefixed relative path with the package name as the
// description. Falls back to an empty action if pnpm is unavailable.
//
//	./packages/widget (@scope/widget)
//	./packages/shared (shared)
//	./apps/web         (web-app)
func ActionWorkspacePaths() carapace.Action {
	return carapace.ActionExecCommandE("pnpm", "list", "--json", "-r", "--depth", "-1")(func(output []byte, err error) carapace.Action {
		var entries []pnpmListEntry
		if len(output) == 0 || json.Unmarshal(output, &entries) != nil {
			return carapace.ActionValues()
		}
		cwd, _ := os.Getwd()
		vals := make([]string, 0, len(entries)*2)
		for _, e := range entries {
			if e.Path == "" {
				continue
			}
			rel, err := filepath.Rel(cwd, e.Path)
			if err != nil {
				continue
			}
			if !strings.HasPrefix(rel, ".") {
				rel = "./" + rel
			}
			desc := e.Name
			if desc == "" {
				desc = "(root)"
			}
			vals = append(vals, rel, desc)
		}
		return carapace.ActionValuesDescribed(vals...).Tag("workspace path").StyleF(style.ForPath).UidF(Uid("workspace-path"))
	}).Cache(0)
}
