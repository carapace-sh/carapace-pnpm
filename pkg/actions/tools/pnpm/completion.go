package pnpm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/carapace-sh/carapace"
	pnpmparser "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
	"github.com/carapace-sh/carapace/pkg/style"
)

// ActionFilters completes pnpm filter selectors (the argument to --filter).
//
// The filter grammar is a comma-separated list of selectors. Each selector is
// an optional "!" negation, an optional prefix "..." (dependents) with "^"
// (exclude self), a base (package name, path, {brace}, [diff], "."), and an
// optional suffix "^..." (dependencies, exclude self) or "..." (dependencies).
//
// This action parses the partial input and, based on the completion context,
// offers package names, paths, brace/diff selectors, the "." self selector,
// relational modifiers, or the "," separator as appropriate.
func ActionFilters() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		expr := c.Value
		ctx := pnpmparser.ParseForCompletion(expr)

		typedPrefix := ""
		partialToken := expr
		if lastComma := strings.LastIndex(expr, ","); lastComma >= 0 {
			typedPrefix = expr[:lastComma+1]
			partialToken = expr[lastComma+1:]
		}
		if i := strings.LastIndex(partialToken, " "); i >= 0 {
			typedPrefix += partialToken[:i+1]
			partialToken = partialToken[i+1:]
		}

		if !strings.Contains(expr, ",") && !strings.Contains(strings.TrimRight(expr, " "), " ") {
			if ctx.AtNewSelector && ctx.Selector != nil && ctx.Selector.PartialBase == "" {
				typedPrefix = ""
				partialToken = expr
			}
		}

		c.Value = partialToken
		return actionForCompletionContext(ctx).Invoke(c).Prefix(typedPrefix).ToA()
	})
}

func actionForCompletionContext(ctx *pnpmparser.CompletionContext) carapace.Action {
	batch := carapace.Batch()

	if ctx.Selector != nil && !ctx.AtNewSelector {
		if ctx.Selector.PartialBase != "" && !ctx.Selector.HasRelation {
			if hasExpected(ctx, pnpmparser.ExpectedRelationOrEnd) {
				batch = append(batch, ActionRelations())
			}
		}
		if ctx.PartialNegation {
			batch = append(batch, actionPackageBases())
		}
		if ctx.Selector.PartialBase != "" && !ctx.Selector.HasRelation && !hasExpected(ctx, pnpmparser.ExpectedRelationOrEnd) {
			batch = append(batch, actionPackageBases())
		}
		if len(batch) > 0 {
			return batch.ToA()
		}
	}

	if ctx.AtNewSelector || hasExpected(ctx, pnpmparser.ExpectedSelector) {
		batch = append(batch, actionPackageBases())
	}

	if hasExpected(ctx, pnpmparser.ExpectedComma) && !ctx.AtNewSelector {
		batch = append(batch, carapace.ActionValues(",").NoSpace().UidF(Uid("comma")))
	}

	if len(batch) > 0 {
		return batch.ToA()
	}

	return carapace.ActionValues()
}

// actionPackageBases returns the action offering candidate selector bases:
// package names, workspace paths, the "." self selector, path-glob prefixes,
// brace selectors, and diff selectors.
func actionPackageBases() carapace.Action {
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
	return batch.ToA()
}

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

// ActionRelations completes the relational modifiers:
//   - "..." suffix  → the package and its dependencies
//   - "^..." suffix → the package's dependencies, excluding itself
func ActionRelations() carapace.Action {
	return carapace.ActionValuesDescribed(
		"...", "the package and its dependencies",
		"^...", "the package's dependencies, excluding itself",
	).NoSpace().UidF(Uid("relation"))
}

func hasExpected(ctx *pnpmparser.CompletionContext, tok pnpmparser.ExpectedToken) bool {
	return slices.Contains(ctx.ExpectedTokens, tok)
}
