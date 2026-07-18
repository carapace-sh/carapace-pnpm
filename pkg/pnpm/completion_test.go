package pnpm

import (
	"slices"
	"testing"
)

func TestCompletionEmpty(t *testing.T) {
	ctx := ParseForCompletion("")
	if len(ctx.ExpectedTokens) == 0 {
		t.Errorf("empty input: no expected tokens")
	}
	if !ctx.AtNewSelector {
		t.Errorf("empty input: expected AtNewSelector")
	}
}

func TestCompletionPackageName(t *testing.T) {
	ctx := ParseForCompletion("foo")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.PartialBase != "foo" {
		t.Errorf("expected PartialBase foo, got %q", ctx.Selector.PartialBase)
	}
	if ctx.Selector.Kind != KindPackageName {
		t.Errorf("expected KindPackageName, got %v", ctx.Selector.Kind)
	}
}

func TestCompletionAfterBaseExpectsRelationOrEnd(t *testing.T) {
	ctx := ParseForCompletion("foo")
	if !hasExpected(ctx, ExpectedRelationOrEnd) {
		t.Errorf("expected ExpectedRelationOrEnd, got %v", ctx.ExpectedTokens)
	}
	if len(ctx.ValidRelations) == 0 {
		t.Errorf("expected valid relations")
	}
}

func TestCompletionAfterDependenciesSuffix(t *testing.T) {
	ctx := ParseForCompletion("foo...")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if !ctx.Selector.IncludeDependencies {
		t.Errorf("expected IncludeDependencies")
	}
	if !ctx.Selector.HasRelation {
		t.Errorf("expected HasRelation")
	}
	if !hasExpected(ctx, ExpectedComma) {
		t.Errorf("expected ExpectedComma after complete suffix, got %v", ctx.ExpectedTokens)
	}
}

func TestCompletionAfterDependentsPrefix(t *testing.T) {
	ctx := ParseForCompletion("...foo")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if !ctx.Selector.IncludeDependents {
		t.Errorf("expected IncludeDependents")
	}
	if ctx.Selector.PartialBase != "foo" {
		t.Errorf("expected PartialBase foo, got %q", ctx.Selector.PartialBase)
	}
}

func TestCompletionBothRelations(t *testing.T) {
	ctx := ParseForCompletion("...foo...")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if !ctx.Selector.IncludeDependents || !ctx.Selector.IncludeDependencies {
		t.Errorf("expected both relations, got dependents=%v deps=%v",
			ctx.Selector.IncludeDependents, ctx.Selector.IncludeDependencies)
	}
}

func TestCompletionExcludeSelfSuffix(t *testing.T) {
	// foo^... → excludeSelf + includeDependencies
	ctx := ParseForCompletion("foo^...")
	if !ctx.Selector.ExcludeSelf || !ctx.Selector.IncludeDependencies {
		t.Errorf("got excludeSelf=%v deps=%v, want true/true",
			ctx.Selector.ExcludeSelf, ctx.Selector.IncludeDependencies)
	}
}

func TestCompletionExcludeSelfPrefix(t *testing.T) {
	// ...^foo → excludeSelf + includeDependents
	ctx := ParseForCompletion("...^foo")
	if !ctx.Selector.ExcludeSelf || !ctx.Selector.IncludeDependents {
		t.Errorf("got excludeSelf=%v dependents=%v, want true/true",
			ctx.Selector.ExcludeSelf, ctx.Selector.IncludeDependents)
	}
}

func TestCompletionNegation(t *testing.T) {
	ctx := ParseForCompletion("!")
	if !ctx.PartialNegation {
		t.Errorf("expected PartialNegation")
	}
	if !hasExpected(ctx, ExpectedSelector) {
		t.Errorf("expected ExpectedSelector after !, got %v", ctx.ExpectedTokens)
	}
}

func TestCompletionNegatedBase(t *testing.T) {
	ctx := ParseForCompletion("!foo")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if !ctx.Selector.Negated {
		t.Errorf("expected Negated")
	}
	if ctx.Selector.PartialBase != "foo" {
		t.Errorf("expected PartialBase foo, got %q", ctx.Selector.PartialBase)
	}
}

func TestCompletionAfterComma(t *testing.T) {
	ctx := ParseForCompletion("foo,")
	if !ctx.AtNewSelector {
		t.Errorf("expected AtNewSelector after comma")
	}
	if !hasExpected(ctx, ExpectedSelector) {
		t.Errorf("expected ExpectedSelector after comma, got %v", ctx.ExpectedTokens)
	}
}

func TestCompletionBetweenSelectors(t *testing.T) {
	ctx := ParseForCompletion("foo, ")
	if !ctx.AtNewSelector {
		t.Errorf("expected AtNewSelector after comma+space")
	}
}

func TestCompletionPath(t *testing.T) {
	ctx := ParseForCompletion("./packages/")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Kind != KindPath {
		t.Errorf("expected KindPath, got %v", ctx.Selector.Kind)
	}
}

func TestCompletionSelfSelector(t *testing.T) {
	ctx := ParseForCompletion(".")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Kind != KindSelf {
		t.Errorf("expected KindSelf, got %v", ctx.Selector.Kind)
	}
}

func TestCompletionDiffSelector(t *testing.T) {
	ctx := ParseForCompletion("[master]")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Kind != KindDiff {
		t.Errorf("expected KindDiff, got %v", ctx.Selector.Kind)
	}
}

func TestCompletionBraceSelector(t *testing.T) {
	ctx := ParseForCompletion("{foo}")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Kind != KindBrace {
		t.Errorf("expected KindBrace, got %v", ctx.Selector.Kind)
	}
}

func TestCompletionAllExpectedTokensNonEmpty(t *testing.T) {
	cases := []string{
		"",
		"foo",
		"foo...",
		"foo^...",
		"!",
		"!foo",
		"foo,",
		"foo, ",
		"./packages/*",
		"@scope/bar",
		".",
		"foo,bar",
		"...foo",
		"...foo...",
		"[master]",
		"{foo}",
	}
	for _, c := range cases {
		ctx := ParseForCompletion(c)
		if len(ctx.ExpectedTokens) == 0 {
			t.Errorf("completion %q: no expected tokens", c)
		}
	}
}

func hasExpected(ctx *CompletionContext, tok ExpectedToken) bool {
	return slices.Contains(ctx.ExpectedTokens, tok)
}
