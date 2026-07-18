package pnpm

import "testing"

// Test cases derived from pnpm's own parse_project_selector test fixtures
// (workspace-projects-filter/src/parse_project_selector/tests.rs and the
// TypeScript parseProjectSelector.ts), cross-referenced with the filtering
// docs. Each asserts the AST shape that pnpm's parser produces.

func TestParsePlainName(t *testing.T) {
	f := mustParse(t, "foo")
	sel := f.Selectors[0]
	if sel.Kind != KindPackageName || sel.Name != "foo" {
		t.Errorf("got kind=%v name=%q, want KindPackageName/foo", sel.Kind, sel.Name)
	}
	assertNoRelations(t, sel)
}

func TestParseScopedName(t *testing.T) {
	f := mustParse(t, "@scope/bar")
	sel := f.Selectors[0]
	if sel.Kind != KindPackageName || sel.Name != "@scope/bar" {
		t.Errorf("got kind=%v name=%q, want KindPackageName/@scope/bar", sel.Kind, sel.Name)
	}
}

func TestParseNameWithDependencies(t *testing.T) {
	f := mustParse(t, "foo...")
	sel := f.Selectors[0]
	if sel.Name != "foo" {
		t.Errorf("name=%q, want foo", sel.Name)
	}
	if !sel.IncludeDependencies || sel.IncludeDependents {
		t.Errorf("relations: deps=%v dependents=%v, want deps=true dependents=false",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
	if sel.ExcludeSelf {
		t.Errorf("excludeSelf=true, want false")
	}
}

func TestParseNameWithDependents(t *testing.T) {
	f := mustParse(t, "...foo")
	sel := f.Selectors[0]
	if sel.Name != "foo" {
		t.Errorf("name=%q, want foo", sel.Name)
	}
	if !sel.IncludeDependents || sel.IncludeDependencies {
		t.Errorf("relations: deps=%v dependents=%v, want deps=false dependents=true",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
}

func TestParseNameWithDependenciesAndDependents(t *testing.T) {
	f := mustParse(t, "...foo...")
	sel := f.Selectors[0]
	if sel.Name != "foo" {
		t.Errorf("name=%q, want foo", sel.Name)
	}
	if !sel.IncludeDependencies || !sel.IncludeDependents {
		t.Errorf("relations: deps=%v dependents=%v, want both true",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
	if sel.ExcludeSelf {
		t.Errorf("excludeSelf=true, want false")
	}
}

func TestParseNameWithDependenciesExcludingSelf(t *testing.T) {
	// foo^... → includeDependencies + excludeSelf
	f := mustParse(t, "foo^...")
	sel := f.Selectors[0]
	if !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("got deps=%v excludeSelf=%v, want deps=true excludeSelf=true",
			sel.IncludeDependencies, sel.ExcludeSelf)
	}
	if sel.IncludeDependents {
		t.Errorf("dependents=true, want false")
	}
}

func TestParseNameWithDependentsExcludingSelf(t *testing.T) {
	// ...^foo → includeDependents + excludeSelf
	f := mustParse(t, "...^foo")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.ExcludeSelf {
		t.Errorf("got dependents=%v excludeSelf=%v, want dependents=true excludeSelf=true",
			sel.IncludeDependents, sel.ExcludeSelf)
	}
	if sel.IncludeDependencies {
		t.Errorf("dependencies=true, want false")
	}
}

func TestParseBothRelationsWithExcludeSelf(t *testing.T) {
	// ...^foo^... → dependents + dependencies + excludeSelf
	f := mustParse(t, "...^foo^...")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("got dependents=%v deps=%v excludeSelf=%v, want all true",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

func TestParseNegation(t *testing.T) {
	f := mustParse(t, "!foo")
	sel := f.Selectors[0]
	if !sel.Negated {
		t.Errorf("expected negated")
	}
	if sel.Name != "foo" {
		t.Errorf("name=%q, want foo", sel.Name)
	}
}

func TestParseNegatedRelation(t *testing.T) {
	f := mustParse(t, "!foo...")
	sel := f.Selectors[0]
	if !sel.Negated || !sel.IncludeDependencies || sel.Name != "foo" {
		t.Errorf("got negated=%v deps=%v name=%q, want true/true/foo",
			sel.Negated, sel.IncludeDependencies, sel.Name)
	}
}

func TestParseSelfSelector(t *testing.T) {
	f := mustParse(t, ".")
	sel := f.Selectors[0]
	if sel.Kind != KindSelf || sel.Path != "." {
		t.Errorf("got kind=%v path=%q, want KindSelf/.", sel.Kind, sel.Path)
	}
}

func TestParseParentSelector(t *testing.T) {
	f := mustParse(t, "..")
	sel := f.Selectors[0]
	if sel.Kind != KindParent || sel.Path != ".." {
		t.Errorf("got kind=%v path=%q, want KindParent/..", sel.Kind, sel.Path)
	}
}

func TestParseRelativePath(t *testing.T) {
	f := mustParse(t, "./foo")
	sel := f.Selectors[0]
	if sel.Kind != KindPath || sel.Path != "./foo" {
		t.Errorf("got kind=%v path=%q, want KindPath/./foo", sel.Kind, sel.Path)
	}
}

func TestParseParentRelativePath(t *testing.T) {
	f := mustParse(t, "../foo")
	sel := f.Selectors[0]
	if sel.Kind != KindPath || sel.Path != "../foo" {
		t.Errorf("got kind=%v path=%q, want KindPath/../foo", sel.Kind, sel.Path)
	}
}

func TestParsePathGlob(t *testing.T) {
	cases := []string{"./packages/*", "./apps/**", "../shared"}
	for _, c := range cases {
		f := mustParse(t, c)
		if f.Selectors[0].Kind != KindPath {
			t.Errorf("parse %q: got %v, want KindPath", c, f.Selectors[0].Kind)
		}
	}
}

func TestParseDiffSelector(t *testing.T) {
	f := mustParse(t, "[master]")
	sel := f.Selectors[0]
	if sel.Kind != KindDiff || sel.Diff != "master" {
		t.Errorf("got kind=%v diff=%q, want KindDiff/master", sel.Kind, sel.Diff)
	}
	assertNoRelations(t, sel)
}

func TestParseDiffWithDependencies(t *testing.T) {
	f := mustParse(t, "[master]...")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependencies {
		t.Errorf("got diff=%q deps=%v, want master/true", sel.Diff, sel.IncludeDependencies)
	}
}

func TestParseDiffWithDependents(t *testing.T) {
	f := mustParse(t, "...[master]")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependents {
		t.Errorf("got diff=%q dependents=%v, want master/true", sel.Diff, sel.IncludeDependents)
	}
}

func TestParseDiffBothRelations(t *testing.T) {
	f := mustParse(t, "...[master]...")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependents || !sel.IncludeDependencies {
		t.Errorf("got diff=%q dependents=%v deps=%v, want master/true/true",
			sel.Diff, sel.IncludeDependents, sel.IncludeDependencies)
	}
}

func TestParseBraceSelector(t *testing.T) {
	f := mustParse(t, "{foo}")
	sel := f.Selectors[0]
	if sel.Kind != KindBrace || sel.BraceInner != "foo" {
		t.Errorf("got kind=%v braceInner=%q, want KindBrace/foo", sel.Kind, sel.BraceInner)
	}
}

func TestParseBraceWithPath(t *testing.T) {
	f := mustParse(t, "{./foo}")
	sel := f.Selectors[0]
	if sel.Kind != KindBrace || sel.BraceInner != "./foo" {
		t.Errorf("got kind=%v braceInner=%q, want KindBrace/./foo", sel.Kind, sel.BraceInner)
	}
}

func TestParseNameBraceAndDiff(t *testing.T) {
	// pattern{foo}[master] → name=pattern, braceInner=foo, diff=master
	f := mustParse(t, "pattern{foo}[master]")
	sel := f.Selectors[0]
	if sel.Name != "pattern" || sel.BraceInner != "foo" || sel.Diff != "master" {
		t.Errorf("got name=%q brace=%q diff=%q, want pattern/foo/master",
			sel.Name, sel.BraceInner, sel.Diff)
	}
}

func TestParseBraceAndDiff(t *testing.T) {
	// {foo}[master] → braceInner=foo, diff=master, no name
	f := mustParse(t, "{foo}[master]")
	sel := f.Selectors[0]
	if sel.Name != "" || sel.BraceInner != "foo" || sel.Diff != "master" {
		t.Errorf("got name=%q brace=%q diff=%q, want empty/foo/master",
			sel.Name, sel.BraceInner, sel.Diff)
	}
}

func TestParseDependentsOfBrace(t *testing.T) {
	// ...{./foo} → includeDependents, braceInner=./foo
	f := mustParse(t, "...{./foo}")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || sel.BraceInner != "./foo" {
		t.Errorf("got dependents=%v braceInner=%q, want true/./foo",
			sel.IncludeDependents, sel.BraceInner)
	}
}

func TestParseMultipleSelectors(t *testing.T) {
	f, err := Parse("foo,bar,baz")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 3 {
		t.Fatalf("got %d selectors, want 3", len(f.Selectors))
	}
	want := []string{"foo", "bar", "baz"}
	for i, w := range want {
		if f.Selectors[i].Name != w {
			t.Errorf("selector %d: name=%q, want %q", i, f.Selectors[i].Name, w)
		}
	}
}

func TestParseWhitespaceAroundComma(t *testing.T) {
	f, err := Parse("@scope/a, @scope/b")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(f.Selectors) != 2 {
		t.Fatalf("got %d selectors, want 2", len(f.Selectors))
	}
	if f.Selectors[0].Name != "@scope/a" || f.Selectors[1].Name != "@scope/b" {
		t.Errorf("names=%q %q, want @scope/a @scope/b",
			f.Selectors[0].Name, f.Selectors[1].Name)
	}
}

func TestParseGlobGroupSingleSelector(t *testing.T) {
	// pnpm docs example: {./apps/*,./packages/*} is a single glob selector.
	f := mustParse(t, "{./apps/*,./packages/*}")
	if len(f.Selectors) != 1 {
		t.Fatalf("got %d selectors, want 1 (comma is inside braces)", len(f.Selectors))
	}
	if f.Selectors[0].BraceInner != "./apps/*,./packages/*" {
		t.Errorf("braceInner=%q, want ./apps/*,./packages/*", f.Selectors[0].BraceInner)
	}
}

func TestParseGlobWithExclusion(t *testing.T) {
	f := mustParse(t, "{./apps/*,!./apps/legacy}")
	if len(f.Selectors) != 1 {
		t.Fatalf("got %d selectors, want 1", len(f.Selectors))
	}
	if f.Selectors[0].BraceInner != "./apps/*,!./apps/legacy" {
		t.Errorf("braceInner=%q", f.Selectors[0].BraceInner)
	}
}

func TestParseComplexRealistic(t *testing.T) {
	cases := []string{
		"./packages/app...,./packages/widget",
		"{./apps/*}, !./apps/legacy",
		"react, react-dom",
		"foo..., !bar",
		"...foo...",
	}
	for _, c := range cases {
		if _, err := Parse(c); err != nil {
			t.Errorf("parse %q: %v", c, err)
		}
	}
}

// --- error cases ---

func TestParseErrorEmpty(t *testing.T) {
	if _, err := Parse(""); err == nil {
		t.Errorf("expected error for empty input")
	}
}

func TestParseErrorNegationOnly(t *testing.T) {
	if _, err := Parse("!"); err == nil {
		t.Errorf("expected error for lone '!'")
	}
}

func TestParseErrorTrailingComma(t *testing.T) {
	if _, err := Parse("foo,"); err == nil {
		t.Errorf("expected error for trailing comma")
	}
}

func TestParseErrorSpaceInSelector(t *testing.T) {
	if _, err := Parse("foo bar"); err == nil {
		t.Errorf("expected error for space within a selector")
	}
}

func TestParseErrorBangInName(t *testing.T) {
	for _, c := range []string{"foo!", "foo!bar"} {
		if _, err := Parse(c); err == nil {
			t.Errorf("expected error for %q (package names cannot contain '!')", c)
		}
	}
}

func TestParseErrorUnterminatedBrace(t *testing.T) {
	if _, err := Parse("{foo"); err == nil {
		t.Errorf("expected error for unterminated '{'")
	}
}

func TestParseErrorUnterminatedBracket(t *testing.T) {
	if _, err := Parse("[master"); err == nil {
		t.Errorf("expected error for unterminated '['")
	}
}

// --- helpers ---

func mustParse(t *testing.T, input string) *Filter {
	t.Helper()
	f, err := Parse(input)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	if len(f.Selectors) != 1 {
		t.Fatalf("parse %q: got %d selectors, want 1", input, len(f.Selectors))
	}
	return f
}

func assertNoRelations(t *testing.T, sel Selector) {
	t.Helper()
	if sel.IncludeDependents || sel.IncludeDependencies || sel.ExcludeSelf {
		t.Errorf("expected no relations, got dependents=%v deps=%v excludeSelf=%v",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}
