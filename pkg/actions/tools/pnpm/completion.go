package pnpm

import (
	"slices"
	"strings"

	"github.com/carapace-sh/carapace"
	pnpmparser "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
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
// package names, the "." self selector, path-glob prefixes, brace selectors,
// and diff selectors.
func actionPackageBases() carapace.Action {
	batch := carapace.Batch(
		carapace.ActionValuesDescribed(
			".", "the package in the current directory",
		).Tag("self").UidF(Uid("base", "kind", "self")),
		carapace.ActionValuesDescribed(
			"./", "packages matching a path glob",
			"{./", "glob group, e.g. {./apps/*,./packages/*}",
			"[", "packages changed since a git ref, e.g. [master]",
		).NoSpace().Tag("path glob").UidF(Uid("base", "kind", "path")),
		ActionWorkspacePackages().Tag("workspace package").UidF(Uid("base", "kind", "name")),
	)
	return batch.ToA()
}

// ActionWorkspacePackages completes workspace package names from
// pnpm-workspace.yaml. The actual list is workspace-specific; this is a stub
// that callers can override. For now it offers an empty action that the
// completer host fills in.
func ActionWorkspacePackages() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		return carapace.ActionValues()
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

func hasExpected(ctx *pnpmparser.CompletionContext, tok pnpmparser.ExpectedToken) bool {
	return slices.Contains(ctx.ExpectedTokens, tok)
}
