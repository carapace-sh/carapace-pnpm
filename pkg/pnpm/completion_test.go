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

func TestCompletionAfterDependentsSuffix(t *testing.T) {
	ctx := ParseForCompletion("foo...")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if !ctx.Selector.HasRelation {
		t.Errorf("expected HasRelation")
	}
	if ctx.Selector.Relation != RelDependents {
		t.Errorf("expected RelDependents, got %v", ctx.Selector.Relation)
	}
	// After a complete selector+suffix, a comma or new selector is valid.
	if !hasExpected(ctx, ExpectedComma) {
		t.Errorf("expected ExpectedComma, got %v", ctx.ExpectedTokens)
	}
}

func TestCompletionAfterDirectDependenciesSuffix(t *testing.T) {
	ctx := ParseForCompletion("foo^...")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Relation != RelDirectDependencies {
		t.Errorf("expected RelDirectDependencies, got %v", ctx.Selector.Relation)
	}
}

func TestCompletionDependenciesPrefix(t *testing.T) {
	ctx := ParseForCompletion("...foo")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Relation != RelDependencies {
		t.Errorf("expected RelDependencies, got %v", ctx.Selector.Relation)
	}
	if ctx.Selector.PartialBase != "foo" {
		t.Errorf("expected PartialBase foo, got %q", ctx.Selector.PartialBase)
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

func TestCompletionPathGlob(t *testing.T) {
	ctx := ParseForCompletion("./packages/")
	if ctx.Selector == nil {
		t.Fatalf("expected non-nil Selector")
	}
	if ctx.Selector.Kind != KindPathGlob {
		t.Errorf("expected KindPathGlob, got %v", ctx.Selector.Kind)
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
