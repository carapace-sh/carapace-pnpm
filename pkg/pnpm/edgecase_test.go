package pnpm

import "testing"

// Edge-case parse tests for pnpm filter selectors, derived from pnpm's own
// parse_project_selector test fixtures and the filtering docs. These exercise
// the parser against tricky combinations to catch regressions.

func TestEdgeCaseBothRelationsAroundName(t *testing.T) {
	// ...foo... — dependents + dependencies, no excludeSelf
	f := mustParse(t, "...foo...")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.IncludeDependencies || sel.ExcludeSelf {
		t.Errorf("got dependents=%v deps=%v excludeSelf=%v, want true/true/false",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

func TestEdgeCaseCaretOnBothSides(t *testing.T) {
	// ...^foo^... — dependents + dependencies + excludeSelf
	f := mustParse(t, "...^foo^...")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("got dependents=%v deps=%v excludeSelf=%v, want all true",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

func TestEdgeCaseCaretOnlyOnPrefix(t *testing.T) {
	// ...^foo — dependents + excludeSelf, no dependencies
	f := mustParse(t, "...^foo")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.ExcludeSelf || sel.IncludeDependencies {
		t.Errorf("got dependents=%v deps=%v excludeSelf=%v, want true/false/true",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

func TestEdgeCaseCaretOnlyOnSuffix(t *testing.T) {
	// foo^... — dependencies + excludeSelf, no dependents
	f := mustParse(t, "foo^...")
	sel := f.Selectors[0]
	if sel.IncludeDependents || !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("got dependents=%v deps=%v excludeSelf=%v, want false/true/true",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

func TestEdgeCaseDiffWithBothRelations(t *testing.T) {
	// ...[master]... — dependents + dependencies + diff
	f := mustParse(t, "...[master]...")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependents || !sel.IncludeDependencies {
		t.Errorf("got diff=%q dependents=%v deps=%v, want master/true/true",
			sel.Diff, sel.IncludeDependents, sel.IncludeDependencies)
	}
}

func TestEdgeCaseNameBraceDiffCombo(t *testing.T) {
	// pattern{foo}[master] — name + brace + diff
	f := mustParse(t, "pattern{foo}[master]")
	sel := f.Selectors[0]
	if sel.Name != "pattern" || sel.BraceInner != "foo" || sel.Diff != "master" {
		t.Errorf("got name=%q brace=%q diff=%q, want pattern/foo/master",
			sel.Name, sel.BraceInner, sel.Diff)
	}
}

func TestEdgeCaseDependentsOfBrace(t *testing.T) {
	// ...{./foo} — dependents of a brace directory
	f := mustParse(t, "...{./foo}")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || sel.BraceInner != "./foo" {
		t.Errorf("got dependents=%v braceInner=%q, want true/./foo",
			sel.IncludeDependents, sel.BraceInner)
	}
}

func TestEdgeCaseNegatedDiff(t *testing.T) {
	// ![master] — exclude changed packages
	f := mustParse(t, "![master]")
	sel := f.Selectors[0]
	if !sel.Negated || sel.Diff != "master" {
		t.Errorf("got negated=%v diff=%q, want true/master", sel.Negated, sel.Diff)
	}
}

func TestEdgeCaseBraceWithExclamationInsideGlob(t *testing.T) {
	// {./apps/*,!./apps/legacy} — ! inside {} is a glob exclusion, not negation
	f := mustParse(t, "{./apps/*,!./apps/legacy}")
	sel := f.Selectors[0]
	if sel.Negated {
		t.Errorf("selector should not be negated (! is inside braces)")
	}
	if sel.BraceInner != "./apps/*,!./apps/legacy" {
		t.Errorf("braceInner=%q, want ./apps/*,!./apps/legacy", sel.BraceInner)
	}
}

func TestEdgeCaseCommaInsideBracesDoesNotSplit(t *testing.T) {
	// {./a,./b} — one selector, not two
	f := mustParse(t, "{./a,./b}")
	if len(f.Selectors) != 1 {
		t.Errorf("expected 1 selector (comma inside braces), got %d", len(f.Selectors))
	}
}

func TestEdgeCaseScopedPackageWithRelations(t *testing.T) {
	// @scope/bar... — scoped package with dependencies suffix
	f := mustParse(t, "@scope/bar...")
	sel := f.Selectors[0]
	if sel.Name != "@scope/bar" || !sel.IncludeDependencies {
		t.Errorf("got name=%q deps=%v, want @scope/bar/true", sel.Name, sel.IncludeDependencies)
	}
}

func TestEdgeCaseNameGlobWithStar(t *testing.T) {
	// @pnpm.e2e/* — name glob
	f := mustParse(t, "@pnpm.e2e/*")
	sel := f.Selectors[0]
	if sel.Kind != KindPackageName || sel.Name != "@pnpm.e2e/*" {
		t.Errorf("got kind=%v name=%q, want KindPackageName/@pnpm.e2e/*",
			sel.Kind, sel.Name)
	}
}

func TestEdgeCaseDotDotParent(t *testing.T) {
	// .. — parent directory
	f := mustParse(t, "..")
	sel := f.Selectors[0]
	if sel.Kind != KindParent || sel.Path != ".." {
		t.Errorf("got kind=%v path=%q, want KindParent/..", sel.Kind, sel.Path)
	}
}

func TestEdgeCaseWindowsStylePath(t *testing.T) {
	// .\packages — Windows-style relative path
	f := mustParse(t, `.\\packages`)
	sel := f.Selectors[0]
	if sel.Kind != KindPath {
		t.Errorf("got kind=%v, want KindPath", sel.Kind)
	}
}

func TestEdgeCaseMultipleSelectorsMixedRelations(t *testing.T) {
	// foo..., !bar, ...^baz
	f, err := Parse("foo..., !bar, ...^baz")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 3 {
		t.Fatalf("expected 3 selectors, got %d", len(f.Selectors))
	}
	if !f.Selectors[0].IncludeDependencies {
		t.Errorf("selector 0: expected IncludeDependencies")
	}
	if !f.Selectors[1].Negated || f.Selectors[1].Name != "bar" {
		t.Errorf("selector 1: expected negated bar, got negated=%v name=%q",
			f.Selectors[1].Negated, f.Selectors[1].Name)
	}
	if !f.Selectors[2].IncludeDependents || !f.Selectors[2].ExcludeSelf {
		t.Errorf("selector 2: expected dependents+excludeSelf, got dependents=%v excludeSelf=%v",
			f.Selectors[2].IncludeDependents, f.Selectors[2].ExcludeSelf)
	}
}

// --- error edge cases ---

func TestEdgeErrorBareDots(t *testing.T) {
	// "..." with no base is a syntax error
	if _, err := Parse("..."); err == nil {
		t.Errorf("expected error for bare '...'")
	}
}

func TestEdgeErrorBareCaretDots(t *testing.T) {
	// "^..." with no base is a syntax error
	if _, err := Parse("^..."); err == nil {
		t.Errorf("expected error for bare '^...'")
	}
}

func TestEdgeErrorDoubleNegation(t *testing.T) {
	// "!!foo" — double negation is not valid
	if _, err := Parse("!!foo"); err == nil {
		t.Errorf("expected error for double negation '!!foo'")
	}
}

func TestEdgeErrorUnterminatedBrace(t *testing.T) {
	if _, err := Parse("{foo"); err == nil {
		t.Errorf("expected error for unterminated '{'")
	}
}

func TestEdgeErrorUnterminatedBracket(t *testing.T) {
	if _, err := Parse("[master"); err == nil {
		t.Errorf("expected error for unterminated '['")
	}
}

func TestEdgeErrorEmptyBrace(t *testing.T) {
	// "{}" — empty braces are not valid (need at least one char inside)
	if _, err := Parse("{}"); err == nil {
		t.Errorf("expected error for empty braces '{}'")
	}
}

func TestEdgeErrorEmptyBracket(t *testing.T) {
	// "[]" — empty brackets are not valid
	if _, err := Parse("[]"); err == nil {
		t.Errorf("expected error for empty brackets '[]'")
	}
}
