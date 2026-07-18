package pnpm

import (
	"slices"
	"strings"
)

// ParseForCompletion parses a partial pnpm filter selector string and returns
// a CompletionContext describing what is expected at the end of the input.
// Partial selectors are allowed — the parser recovers from incomplete input to
// report what tokens would be valid at the cursor position (end of input).
func ParseForCompletion(input string) *CompletionContext {
	ctx := &CompletionContext{}

	// Find the last top-level selector (the one being completed). Split on
	// top-level commas, respecting brace/bracket depth.
	last := lastSelector(input)
	if last == "" {
		ctx.ExpectedTokens = append(ctx.ExpectedTokens, ExpectedSelector)
		ctx.AtNewSelector = true
		return ctx
	}

	selCtx := &SelectorContext{}
	ctx.Selector = selCtx

	// Determine whether anything was typed before this selector (a comma or
	// other content). If so, the cursor is not at a "new selector" position
	// in the sense of the very start — but the in-progress selector is new.
	trimmed := strings.TrimRight(strings.TrimRight(input, " \t\r\n"), ",")
	ctx.AtNewSelector = last == input || strings.HasSuffix(trimmed, ",")

	body := last

	// Negation.
	if strings.HasPrefix(body, "!") {
		selCtx.Negated = true
		body = body[1:]
		if body == "" {
			ctx.ExpectedTokens = append(ctx.ExpectedTokens, ExpectedSelector)
			ctx.PartialNegation = true
			ctx.AtNewSelector = true
			return ctx
		}
	}

	// Strip trailing "..." (dependencies) and optional preceding "^".
	if strings.HasSuffix(body, "...") {
		selCtx.IncludeDependencies = true
		selCtx.HasRelation = true
		body = body[:len(body)-3]
		if strings.HasSuffix(body, "^") {
			selCtx.ExcludeSelf = true
			body = body[:len(body)-1]
		}
	}

	// Strip leading "..." (dependents) and optional following "^".
	if strings.HasPrefix(body, "...") {
		selCtx.IncludeDependents = true
		selCtx.HasRelation = true
		body = body[3:]
		if strings.HasPrefix(body, "^") {
			selCtx.ExcludeSelf = true
			body = body[1:]
		}
	}

	// body is now the base (possibly partial). If empty, the user typed only
	// relations so far — but that's not a valid selector on its own unless
	// it's a bare "[ref]" or "." etc. Expect a base.
	selCtx.PartialBase = body
	selCtx.Kind = classifyPartialBase(body)

	// If the base is empty after stripping relations, expect a selector.
	if body == "" {
		ctx.ExpectedTokens = append(ctx.ExpectedTokens, ExpectedSelector)
		ctx.AtNewSelector = true
		return ctx
	}

	// If relations were already consumed (suffix present), the selector is
	// complete — a comma or end is valid.
	if selCtx.IncludeDependencies {
		ctx.ExpectedTokens = append(ctx.ExpectedTokens, ExpectedComma, ExpectedSelector)
		ctx.AtNewSelector = true
		return ctx
	}

	// Base present, no trailing relation yet — a relation or end is valid.
	ctx.ExpectedTokens = append(ctx.ExpectedTokens, ExpectedRelationOrEnd)
	ctx.ValidRelations = append(ctx.ValidRelations,
		ValidRelation{Op: "...", Description: "the package and its dependencies"},
		ValidRelation{Op: "^...", Description: "the package's dependencies, excluding itself"},
	)

	ctx.ExpectedTokens = dedupTokens(ctx.ExpectedTokens)
	ctx.ValidRelations = dedupRelations(ctx.ValidRelations)
	return ctx
}

// lastSelector returns the text of the final top-level selector in the input
// (the one in progress at the cursor). Returns "" if the input is empty or
// only whitespace/commas.
func lastSelector(input string) string {
	// Walk the input tracking brace/bracket depth; split on top-level commas.
	depth := 0
	lastStart := 0
	for i := 0; i < len(input); i++ {
		r := input[i]
		switch r {
		case '{', '[':
			depth++
		case '}', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				lastStart = i + 1
			}
		}
	}
	s := input[lastStart:]
	// Trim leading whitespace.
	s = strings.TrimLeft(s, " \t\r\n")
	return s
}

// classifyPartialBase classifies the (partial) base text for completion.
func classifyPartialBase(s string) SelectorKind {
	if s == "" {
		return 0
	}
	if s == "." {
		return KindSelf
	}
	if s == ".." {
		return KindParent
	}
	if isLocation(s) {
		return KindPath
	}
	if strings.HasPrefix(s, "{") {
		return KindBrace
	}
	if strings.HasPrefix(s, "[") {
		return KindDiff
	}
	return KindPackageName
}

// --- dedup helpers ---

func dedupTokens(toks []ExpectedToken) []ExpectedToken {
	slices.Sort(toks)
	return slices.Compact(toks)
}

func dedupRelations(rels []ValidRelation) []ValidRelation {
	if len(rels) <= 1 {
		return rels
	}
	seen := map[string]bool{}
	out := rels[:0]
	for _, r := range rels {
		if seen[r.Op] {
			continue
		}
		seen[r.Op] = true
		out = append(out, r)
	}
	return out
}
