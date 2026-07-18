package pnpm

import (
	"slices"
	"strings"

	"github.com/carapace-sh/carapace"
	pnpmparser "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
)

// ActionFilters completes pnpm filter selectors (the argument to --filter).
//
// The filter grammar is a comma-separated list of selectors, each optionally
// negated with "!", each naming a package (by name or path glob) and
// optionally suffixed with "..." (dependents) or "^..." (dependencies).
//
// This action parses the partial input and, based on the completion context,
// offers package names, path globs, the "." selector, relational suffixes, or
// the "," separator as appropriate.
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
		// Strip trailing whitespace from the prefix so partialToken is the
		// in-progress selector.
		if i := strings.LastIndex(partialToken, " "); i >= 0 {
			typedPrefix += partialToken[:i+1]
			partialToken = partialToken[i+1:]
		}

		// If the whole input is a single in-progress selector with no commas,
		// treat the entire input as the partial token (no prefix).
		if !strings.Contains(expr, ",") && !strings.Contains(strings.TrimRight(expr, " "), " ") {
			if ctx.AtNewSelector && ctx.Selector != nil && ctx.Selector.PartialBase == "" {
				// Cursor at start of selector.
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
		// Inside a selector that already has a base: offer relational suffixes.
		if ctx.Selector.PartialBase != "" && !ctx.Selector.HasRelation {
			if hasExpected(ctx, pnpmparser.ExpectedRelationOrEnd) {
				batch = append(batch,
					carapace.ActionValuesDescribed(
						"...", "the package and its dependents",
						"^...", "the package and its dependencies",
					).NoSpace().UidF(Uid("relation")),
				)
			}
		}
		// After a negation with no base: offer package selectors.
		if ctx.PartialNegation {
			batch = append(batch, actionPackageBases())
		}
		// If we have a partial base and no relation yet, also offer more
		// package-name candidates (prefix completion).
		if ctx.Selector.PartialBase != "" && !ctx.Selector.HasRelation && !hasExpected(ctx, pnpmparser.ExpectedRelationOrEnd) {
			batch = append(batch, actionPackageBases())
		}
		if len(batch) > 0 {
			return batch.ToA()
		}
	}

	// At a new selector position (start, or after a comma): offer bases.
	if ctx.AtNewSelector || hasExpected(ctx, pnpmparser.ExpectedSelector) {
		batch = append(batch, actionPackageBases())
	}

	// After a complete selector, a comma is valid.
	if hasExpected(ctx, pnpmparser.ExpectedComma) && !ctx.AtNewSelector {
		batch = append(batch, carapace.ActionValues(",").NoSpace().UidF(Uid("comma")))
	}

	if len(batch) > 0 {
		return batch.ToA()
	}

	return carapace.ActionValues()
}

// actionPackageBases returns the action offering candidate selector bases:
// package names, the "." self selector, and path-glob prefixes.
func actionPackageBases() carapace.Action {
	batch := carapace.Batch(
		carapace.ActionValuesDescribed(
			".", "the package in the current directory",
		).Tag("self").UidF(Uid("base", "kind", "self")),
		carapace.ActionValuesDescribed(
			"./", "packages matching a path glob",
			"{./", "glob group, e.g. {./apps/*,./packages/*}",
		).NoSpace().Tag("path glob").UidF(Uid("base", "kind", "path")),
		ActionWorkspacePackages().Tag("workspace package").UidF(Uid("base", "kind", "name")),
	)
	return batch.ToA()
}

// ActionWorkspacePackages completes workspace package names from
// pnpm-workspace.yaml. The actual list is workspace-specific; this is a stub
// that callers can override via [carapace.Action]. For now it offers a few
// common example names so the completion is non-empty in the absence of a
// workspace to introspect.
func ActionWorkspacePackages() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		// In a real integration this would parse pnpm-workspace.yaml and list
		// package names. The lexer package is intentionally pure (no I/O), so
		// the action layer is the right place for that lookup. For now, expose
		// an empty-ish action that the completer host fills in.
		return carapace.ActionValues()
	})
}

// ActionRelations completes the relational suffixes "..." and "^...".
func ActionRelations() carapace.Action {
	return carapace.ActionValuesDescribed(
		"...", "the package and its dependents",
		"^...", "the package and its dependencies",
	).NoSpace().UidF(Uid("relation"))
}

func hasExpected(ctx *pnpmparser.CompletionContext, tok pnpmparser.ExpectedToken) bool {
	return slices.Contains(ctx.ExpectedTokens, tok)
}
