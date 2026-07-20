package pnpm

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/carapace-sh/carapace"
	pnpmlexer "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
)

// ActionScripts completes scripts from the current package's package.json
//
//	build
//	test
func ActionScripts() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if pj, err := loadPackageJson(c); err != nil {
			return carapace.ActionMessage(err.Error())
		} else {
			vals := make([]string, 0)
			for name := range pj.Scripts {
				vals = append(vals, name)
			}
			return carapace.ActionValues(vals...)
		}
	})
}

// ActionScriptsForFilter completes scripts scoped by a --filter value.
//
// If the filter is a simple package name (no relations, no globs, no diff),
// only that package's scripts are offered. If the filter is empty or complex
// (relations, globs, negation, diff), falls back to the union of all workspace
// scripts (via ActionWorkspaceScripts) — since resolving those requires the
// full workspace graph.
func ActionScriptsForFilter(filter string) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		if filter == "" {
			return carapace.Batch(
				ActionScripts(),
				ActionWorkspaceScripts(),
			).ToA()
		}

		f, err := pnpmlexer.Parse(filter)
		if err != nil {
			return carapace.Batch(
				ActionScripts(),
				ActionWorkspaceScripts(),
			).ToA()
		}

		var pkgNames []string
		hasComplex := false
		for _, sel := range f.Selectors {
			if sel.Negated || sel.IncludeDependents || sel.IncludeDependencies ||
				sel.ExcludeSelf || sel.Diff != "" || sel.BraceInner != "" ||
				sel.Kind == pnpmlexer.KindPath || sel.Kind == pnpmlexer.KindSelf ||
				sel.Kind == pnpmlexer.KindParent {
				hasComplex = true
				continue
			}
			if sel.Name != "" {
				pkgNames = append(pkgNames, sel.Name)
			}
		}

		if hasComplex {
			return ActionWorkspaceScripts()
		}

		if len(pkgNames) == 0 {
			return carapace.Batch(
				ActionScripts(),
				ActionWorkspaceScripts(),
			).ToA()
		}

		return actionScriptsForPackages(pkgNames)
	})
}

// actionScriptsForPackages reads the scripts from the package.json files of
// the named workspace packages, resolved via `pnpm ls --json -r --depth -1`.
func actionScriptsForPackages(names []string) carapace.Action {
	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}
	return carapace.ActionExecCommandE("pnpm", "ls", "--json", "-r", "--depth", "-1")(func(output []byte, err error) carapace.Action {
		if err != nil && len(output) == 0 {
			return ActionWorkspaceScripts()
		}
		var entries []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		}
		if err := json.Unmarshal(output, &entries); err != nil {
			return ActionWorkspaceScripts()
		}

		vals := make([]string, 0)
		seen := make(map[string]bool)
		for _, e := range entries {
			if !nameSet[e.Name] || e.Path == "" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(e.Path, "package.json"))
			if err != nil {
				continue
			}
			var pj packageJson
			if json.Unmarshal(data, &pj) != nil {
				continue
			}
			for name := range pj.Scripts {
				if !seen[name] {
					seen[name] = true
					vals = append(vals, name)
				}
			}
		}
		if len(vals) == 0 {
			return ActionWorkspaceScripts()
		}
		return carapace.ActionValues(vals...).Tag("scripts")
	})
}
