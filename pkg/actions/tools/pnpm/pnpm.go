package pnpm

import (
	"strings"

	"github.com/carapace-sh/carapace"
	pnpmlexer "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
)

// ActionFilters completes pnpm filter selectors (the argument to --filter).
//
// Uses the carapace-pnpm lexer for grammar-aware completion of package names,
// relational modifiers, negation, comma-separated lists, and location
// selectors. For [ref] diff selectors, shells out to git to complete branches
// and tags. For {dir} brace selectors, completes directories.
//
//	foo
//	@scope/bar
//	foo...
//	...^foo
//	[master]
//	{./apps/*}
func ActionFilters() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		value := c.Value

		ctx := pnpmlexer.ParseForCompletion(value)
		if ctx.Selector != nil && ctx.Selector.PartialBase != "" {
			partial := ctx.Selector.PartialBase

			// Inside a [ref] — complete git branches and tags.
			if idx := strings.LastIndex(partial, "["); idx != -1 && !strings.Contains(partial[idx+1:], "]") {
				prefix := value[:len(value)-len(partial)+idx+1]
				c.Value = partial[idx+1:]
				return actionGitRefs().Invoke(c).Prefix(prefix).Suffix("]").ToA().NoSpace()
			}

			// Inside a {dir} — complete directories.
			if idx := strings.LastIndex(partial, "{"); idx != -1 && !strings.Contains(partial[idx+1:], "}") {
				prefix := value[:len(value)-len(partial)+idx+1]
				c.Value = partial[idx+1:]
				return carapace.ActionDirectories().Invoke(c).Prefix(prefix).Suffix("}").ToA().NoSpace()
			}
		}

		// Split on the last top-level comma so we complete only the
		// in-progress selector (the part after the last comma).
		typedPrefix := ""
		partialToken := value
		if lastComma := lastTopLevelComma(value); lastComma >= 0 {
			typedPrefix = value[:lastComma+1]
			partialToken = value[lastComma+1:]
		}
		// Strip trailing whitespace from the partial.
		if i := strings.LastIndex(partialToken, " "); i >= 0 {
			typedPrefix += partialToken[:i+1]
			partialToken = partialToken[i+1:]
		}

		origValue := c.Value
		c.Value = partialToken
		result := actionLexerFilters().Invoke(c).Prefix(typedPrefix).ToA()
		c.Value = origValue
		return result
	})
}

// lastTopLevelComma returns the index of the last comma at brace depth 0,
// or -1 if there is none. Commas inside {...} are part of a glob and don't
// separate selectors.
func lastTopLevelComma(s string) int {
	depth := 0
	last := -1
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '{':
			depth++
		case '}':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				last = i
			}
		}
	}
	return last
}

// actionLexerFilters wraps the lexer's ActionFilters (from the same package)
// to offer package names, relational modifiers, paths, and static bases.
func actionLexerFilters() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		batch := carapace.Batch(
			carapace.ActionValuesDescribed(
				".", "the package in the current directory",
			).Tag("self").UidF(Uid("base", "kind", "self")),
			carapace.ActionValuesDescribed(
				"{./", "glob group, e.g. {./apps/*,./packages/*}",
				"[", "packages changed since a git ref, e.g. [master]",
			).NoSpace().Tag("path glob").UidF(Uid("base", "kind", "path")),
			ActionWorkspacePackages().Tag("workspace package").UidF(Uid("base", "kind", "name")),
			ActionWorkspacePaths().Tag("workspace path").UidF(Uid("base", "kind", "workspace-path")),
		)

		// Offer relational modifiers when appropriate.
		ctx := pnpmlexer.ParseForCompletion(c.Value)
		if ctx.Selector != nil && ctx.Selector.PartialBase != "" && !ctx.Selector.HasRelation {
			batch = append(batch, ActionRelations())
		}

		return batch.ToA()
	})
}

// actionGitRefs completes git branches and tags by shelling out to git
// directly — no carapace-bin dependency needed.
//
//	master (Add new feature)
//	v0.0.1 (Release v0.0.1)
func actionGitRefs() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		return carapace.Batch(
			actionGitBranches(),
			actionGitTags(),
		).ToA().UidF(Uid("git-ref"))
	})
}

// actionGitBranches completes local and remote branches with their last
// commit message as the description.
func actionGitBranches() carapace.Action {
	return carapace.ActionExecCommandE("git", "branch", "--all", "--format=%(refname:short)%09%(contents:subject)")(func(output []byte, err error) carapace.Action {
		if err != nil && len(output) == 0 {
			return carapace.ActionValues()
		}
		vals := make([]string, 0)
		for _, line := range strings.Split(string(output), "\n") {
			if line == "" {
				continue
			}
			fields := strings.SplitN(line, "\t", 2)
			if len(fields) == 2 {
				vals = append(vals, fields[0], fields[1])
			} else if fields[0] != "" {
				vals = append(vals, fields[0])
			}
		}
		return carapace.ActionValuesDescribed(vals...).Tag("git branches")
	})
}

// actionGitTags completes git tags with their last commit message.
func actionGitTags() carapace.Action {
	return carapace.ActionExecCommandE("git", "tag", "--format=%(refname:short)%09%(contents:subject)")(func(output []byte, err error) carapace.Action {
		if err != nil && len(output) == 0 {
			return carapace.ActionValues()
		}
		vals := make([]string, 0)
		for _, line := range strings.Split(string(output), "\n") {
			if line == "" {
				continue
			}
			fields := strings.SplitN(line, "\t", 2)
			if len(fields) == 2 {
				vals = append(vals, fields[0], fields[1])
			} else if fields[0] != "" {
				vals = append(vals, fields[0])
			}
		}
		return carapace.ActionValuesDescribed(vals...).Tag("git tags")
	})
}

// ActionDependencyNames completes dependency names from package.json
//
//	express
//	lodash
func ActionDependencyNames() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		pj, err := loadPackageJson(c)
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		vals := make([]string, 0)
		seen := make(map[string]bool)

		for name := range pj.Dependencies {
			if !seen[name] {
				seen[name] = true
				vals = append(vals, name)
			}
		}
		for name := range pj.DevDependencies {
			if !seen[name] {
				seen[name] = true
				vals = append(vals, name)
			}
		}
		for name := range pj.OptionalDependencies {
			if !seen[name] {
				seen[name] = true
				vals = append(vals, name)
			}
		}
		return carapace.ActionValues(vals...).Tag("dependencies")
	})
}

// ActionDependencies completes dependencies with their version from package.json
//
//	express@4.18.0
//	lodash@4.17.21
func ActionDependencies() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		pj, err := loadPackageJson(c)
		if err != nil {
			return carapace.ActionMessage(err.Error())
		}

		vals := make([]string, 0)
		seen := make(map[string]bool)

		addDeps := func(deps map[string]string) {
			for name, version := range deps {
				if !seen[name] {
					seen[name] = true
					vals = append(vals, name, version)
				}
			}
		}

		addDeps(pj.Dependencies)
		addDeps(pj.DevDependencies)
		addDeps(pj.OptionalDependencies)

		return carapace.ActionValuesDescribed(vals...).Tag("dependencies")
	})
}

// ActionRelations completes the relational modifiers:
//   - "..." suffix  → the package and its dependencies
//   - "^..." suffix → the package's dependencies, excluding itself
func ActionRelations() carapace.Action {
	return carapace.ActionValuesDescribed(
		"...", "the package and its dependencies",
		"^...", "the package's dependencies, excluding itself",
	).NoSpace().UidF(Uid("relation"))
}

// ActionStorePath completes store paths
//
//	/home/user/.local/share/pnpm/store/v3
func ActionStorePath() carapace.Action {
	return carapace.ActionExecCommand("pnpm", "store", "path")(func(output []byte) carapace.Action {
		path := strings.TrimSpace(string(output))
		return carapace.ActionValues(path)
	})
}

// ActionStoreStatus completes store status information
//
//	OK packages were found in the store
//	MISSING packages were not found in the store
func ActionStoreStatus() carapace.Action {
	return carapace.ActionExecCommand("pnpm", "store", "status")(func(output []byte) carapace.Action {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		vals := make([]string, 0)
		for _, line := range lines {
			if line != "" {
				vals = append(vals, line)
			}
		}
		return carapace.ActionValues(vals...)
	})
}

// ActionStorePackages completes packages in the store
//
//	express/4.18.0
//	lodash/4.17.21
func ActionStorePackages() carapace.Action {
	return carapace.ActionExecCommand("pnpm", "list", "--parseable", "--depth", "0")(func(output []byte) carapace.Action {
		lines := strings.Split(strings.TrimSpace(string(output)), "\n")
		vals := make([]string, 0)
		for _, line := range lines {
			if line != "" {
				if idx := strings.LastIndex(line, "/"); idx != -1 {
					vals = append(vals, line[idx+1:])
				} else {
					vals = append(vals, line)
				}
			}
		}
		return carapace.ActionValues(vals...).Tag("store packages")
	})
}

// ActionStorePrune completes store prune options
func ActionStorePrune() carapace.Action {
	return carapace.ActionValues()
}
