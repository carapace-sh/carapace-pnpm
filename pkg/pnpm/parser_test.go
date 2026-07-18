package pnpm

import "testing"

// Realistic pnpm --filter selector examples sourced from the pnpm docs.
var parseSuccessCases = []string{
	// Package name selectors
	"foo",
	"@scope/bar",
	"@scope/nested/pkg",

	// Self selector
	".",

	// Path glob selectors
	"./packages/*",
	"../shared",
	"{./apps/*,./packages/*}",
	"packages/**",

	// Relational suffixes and prefix
	"foo...",
	"...foo",
	"foo^...",
	"./packages/app...",

	// Negation
	"!foo",
	"!./packages/legacy",

	// Comma-separated unions
	"foo,bar",
	"foo,bar,baz",
	"@scope/a, @scope/b",
	"foo..., !bar",

	// Complex realistic examples
	"./packages/app...,./packages/widget",
	"{./apps/*}, !./apps/legacy",
	"react, react-dom, !react@17",
}

func TestParseAllSuccessCases(t *testing.T) {
	for _, sel := range parseSuccessCases {
		f, err := Parse(sel)
		if err != nil {
			t.Errorf("parse %q: %v", sel, err)
			continue
		}
		if f == nil {
			t.Errorf("parse %q: nil filter", sel)
			continue
		}
		if len(f.Selectors) == 0 {
			t.Errorf("parse %q: no selectors", sel)
		}
	}
}

func TestParseSinglePackageName(t *testing.T) {
	f, err := Parse("foo")
	if err != nil {
		t.Fatalf("parse foo: %v", err)
	}
	if len(f.Selectors) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(f.Selectors))
	}
	sel := f.Selectors[0]
	if sel.Kind != KindPackageName {
		t.Errorf("expected KindPackageName, got %v", sel.Kind)
	}
	if sel.Base != "foo" {
		t.Errorf("expected base foo, got %q", sel.Base)
	}
	if sel.Negated {
		t.Errorf("expected not negated")
	}
	if sel.Relation != RelNone {
		t.Errorf("expected no relation, got %v", sel.Relation)
	}
}

func TestParseScopedPackage(t *testing.T) {
	f, err := Parse("@scope/bar")
	if err != nil {
		t.Fatalf("parse @scope/bar: %v", err)
	}
	sel := f.Selectors[0]
	if sel.Kind != KindPackageName {
		t.Errorf("expected KindPackageName, got %v", sel.Kind)
	}
	if sel.Base != "@scope/bar" {
		t.Errorf("expected base @scope/bar, got %q", sel.Base)
	}
}

func TestParseSelfSelector(t *testing.T) {
	f, err := Parse(".")
	if err != nil {
		t.Fatalf("parse .: %v", err)
	}
	sel := f.Selectors[0]
	if sel.Kind != KindSelf {
		t.Errorf("expected KindSelf, got %v", sel.Kind)
	}
}

func TestParsePathGlob(t *testing.T) {
	cases := []string{
		"./packages/*",
		"../shared",
		"{./apps/*,./packages/*}",
		"packages/**",
	}
	for _, c := range cases {
		f, err := Parse(c)
		if err != nil {
			t.Errorf("parse %q: %v", c, err)
			continue
		}
		if f.Selectors[0].Kind != KindPathGlob {
			t.Errorf("parse %q: expected KindPathGlob, got %v", c, f.Selectors[0].Kind)
		}
	}
}

func TestParseDependentsSuffix(t *testing.T) {
	f, err := Parse("foo...")
	if err != nil {
		t.Fatalf("parse foo...: %v", err)
	}
	sel := f.Selectors[0]
	if sel.Relation != RelDependents {
		t.Errorf("expected RelDependents, got %v", sel.Relation)
	}
	if sel.RelSpan == nil {
		t.Errorf("expected non-nil RelSpan")
	}
}

func TestParseDependenciesPrefix(t *testing.T) {
	f, err := Parse("...foo")
	if err != nil {
		t.Fatalf("parse ...foo: %v", err)
	}
	sel := f.Selectors[0]
	if sel.Relation != RelDependencies {
		t.Errorf("expected RelDependencies, got %v", sel.Relation)
	}
	if sel.RelSpan == nil {
		t.Errorf("expected non-nil RelSpan")
	}
	if sel.Base != "foo" {
		t.Errorf("expected base foo, got %q", sel.Base)
	}
}

func TestParseDirectDependenciesSuffix(t *testing.T) {
	f, err := Parse("foo^...")
	if err != nil {
		t.Fatalf("parse foo^...: %v", err)
	}
	sel := f.Selectors[0]
	if sel.Relation != RelDirectDependencies {
		t.Errorf("expected RelDirectDependencies, got %v", sel.Relation)
	}
}

func TestParseNegation(t *testing.T) {
	f, err := Parse("!foo")
	if err != nil {
		t.Fatalf("parse !foo: %v", err)
	}
	sel := f.Selectors[0]
	if !sel.Negated {
		t.Errorf("expected negated")
	}
	if sel.NegateSpan == nil {
		t.Errorf("expected non-nil NegateSpan")
	}
	if sel.Base != "foo" {
		t.Errorf("expected base foo, got %q", sel.Base)
	}
}

func TestParseMultipleSelectors(t *testing.T) {
	f, err := Parse("foo,bar,baz")
	if err != nil {
		t.Fatalf("parse foo,bar,baz: %v", err)
	}
	if len(f.Selectors) != 3 {
		t.Fatalf("expected 3 selectors, got %d", len(f.Selectors))
	}
	want := []string{"foo", "bar", "baz"}
	for i, w := range want {
		if f.Selectors[i].Base != w {
			t.Errorf("selector %d: expected base %q, got %q", i, w, f.Selectors[i].Base)
		}
	}
}

func TestParseWhitespaceAroundComma(t *testing.T) {
	f, err := Parse("@scope/a, @scope/b")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(f.Selectors))
	}
	if f.Selectors[0].Base != "@scope/a" {
		t.Errorf("selector 0: expected @scope/a, got %q", f.Selectors[0].Base)
	}
	if f.Selectors[1].Base != "@scope/b" {
		t.Errorf("selector 1: expected @scope/b, got %q", f.Selectors[1].Base)
	}
}

func TestParseErrorEmpty(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Errorf("expected error for empty input")
	}
}

func TestParseErrorNegationOnly(t *testing.T) {
	_, err := Parse("!")
	if err == nil {
		t.Errorf("expected error for lone '!'")
	}
}

func TestParseErrorTrailingComma(t *testing.T) {
	_, err := Parse("foo,")
	if err == nil {
		t.Errorf("expected error for trailing comma")
	}
}

func TestParseErrorUnexpectedToken(t *testing.T) {
	_, err := Parse("foo bar")
	if err == nil {
		t.Errorf("expected error for space within a selector")
	}
}

func TestParseErrorBangInName(t *testing.T) {
	// '!' is the negation operator, not a package-name character.
	for _, c := range []string{"foo!", "foo!bar"} {
		_, err := Parse(c)
		if err == nil {
			t.Errorf("expected error for %q (package names cannot contain '!')", c)
		}
	}
}

func TestParseGlobWithExclusion(t *testing.T) {
	// A comma inside {...} is part of the glob, not a selector separator.
	// '!' inside {...} is a glob exclusion pattern.
	f, err := Parse("{./apps/*,!./apps/legacy}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 1 {
		t.Fatalf("expected 1 selector (glob is one base), got %d", len(f.Selectors))
	}
	if f.Selectors[0].Base != "{./apps/*,!./apps/legacy}" {
		t.Errorf("expected full glob as base, got %q", f.Selectors[0].Base)
	}
	if f.Selectors[0].Kind != KindPathGlob {
		t.Errorf("expected KindPathGlob, got %v", f.Selectors[0].Kind)
	}
}

func TestParseGlobGroupMultipleSelectors(t *testing.T) {
	// pnpm docs example: {./apps/*,./packages/*} is a single glob selector.
	f, err := Parse("{./apps/*,./packages/*}")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(f.Selectors))
	}
}
