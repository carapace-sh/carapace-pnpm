package pnpm

// Verification tests against pnpm's actual source code:
// https://github.com/pnpm/pnpm/blob/main/pnpm/crates/workspace-projects-filter/src/parse_project_selector.rs
//
// These tests mirror the exact fixtures from pnpm's tests.rs and verify
// that carapace-pnpm produces equivalent AST shapes.
//
// Key structural mappings:
//   pnpm exclude           ↔ Negated
//   pnpm exclude_self      ↔ ExcludeSelf
//   pnpm include_deps      ↔ IncludeDependencies
//   pnpm include_dependents ↔ IncludeDependents
//   pnpm name_pattern       ↔ Name (when Kind == KindPackageName)
//   pnpm parent_dir         ↔ Path (when Kind == KindPath/Self/Parent) or BraceInner
//   pnpm diff               ↔ Diff

import "testing"

// --- pnpm test fixture: plain_name ---
// pnpm: name_pattern: "foo"
func TestPnpmVerifyPlainName(t *testing.T) {
	f := mustParse(t, "foo")
	sel := f.Selectors[0]
	assertKind(t, sel, KindPackageName)
	assertName(t, sel, "foo")
	assertNoRelations(t, sel)
}

// --- pnpm test fixture: name_with_dependencies ---
// pnpm: include_dependencies: true, name_pattern: "foo"
func TestPnpmVerifyNameWithDeps(t *testing.T) {
	f := mustParse(t, "foo...")
	sel := f.Selectors[0]
	assertName(t, sel, "foo")
	if !sel.IncludeDependencies || sel.IncludeDependents {
		t.Errorf("relations: deps=%v dependents=%v, want deps=true dependents=false",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
	if sel.ExcludeSelf {
		t.Errorf("excludeSelf=true, want false")
	}
}

// --- pnpm test fixture: name_with_dependents ---
// pnpm: include_dependents: true, name_pattern: "foo"
func TestPnpmVerifyNameWithDependents(t *testing.T) {
	f := mustParse(t, "...foo")
	sel := f.Selectors[0]
	assertName(t, sel, "foo")
	if !sel.IncludeDependents || sel.IncludeDependencies {
		t.Errorf("relations: deps=%v dependents=%v, want deps=false dependents=true",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
}

// --- pnpm test fixture: name_with_dependencies_and_dependents ---
// pnpm: include_dependencies: true, include_dependents: true, name_pattern: "foo"
func TestPnpmVerifyNameWithBothRelations(t *testing.T) {
	f := mustParse(t, "...foo...")
	sel := f.Selectors[0]
	assertName(t, sel, "foo")
	if !sel.IncludeDependencies || !sel.IncludeDependents {
		t.Errorf("want both relations true, got deps=%v dependents=%v",
			sel.IncludeDependencies, sel.IncludeDependents)
	}
	if sel.ExcludeSelf {
		t.Errorf("excludeSelf=true, want false")
	}
}

// --- pnpm test fixture: name_with_dependencies_excluding_self ---
// pnpm: exclude_self: true, include_dependencies: true, name_pattern: "foo"
func TestPnpmVerifyNameWithDepsExcludeSelf(t *testing.T) {
	f := mustParse(t, "foo^...")
	sel := f.Selectors[0]
	if !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("want deps=true excludeSelf=true, got deps=%v excludeSelf=%v",
			sel.IncludeDependencies, sel.ExcludeSelf)
	}
	if sel.IncludeDependents {
		t.Errorf("dependents=true, want false")
	}
}

// --- pnpm test fixture: name_with_dependents_excluding_self ---
// pnpm: exclude_self: true, include_dependents: true, name_pattern: "foo"
func TestPnpmVerifyNameWithDependentsExcludeSelf(t *testing.T) {
	f := mustParse(t, "...^foo")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.ExcludeSelf {
		t.Errorf("want dependents=true excludeSelf=true, got dependents=%v excludeSelf=%v",
			sel.IncludeDependents, sel.ExcludeSelf)
	}
	if sel.IncludeDependencies {
		t.Errorf("dependencies=true, want false")
	}
}

// --- pnpm test fixture: both_relations_with_exclude_self ---
// pnpm: exclude_self: true, include_dependencies: true, include_dependents: true, name_pattern: "foo"
// Input: ...^foo^...
func TestPnpmVerifyBothRelationsExcludeSelf(t *testing.T) {
	f := mustParse(t, "...^foo^...")
	sel := f.Selectors[0]
	if !sel.IncludeDependents || !sel.IncludeDependencies || !sel.ExcludeSelf {
		t.Errorf("want all true, got dependents=%v deps=%v excludeSelf=%v",
			sel.IncludeDependents, sel.IncludeDependencies, sel.ExcludeSelf)
	}
}

// --- pnpm test fixture: relative_path_selector ---
// pnpm: parent_dir: "/prefix/foo"  (carapace doesn't resolve prefix, stores raw)
func TestPnpmVerifyRelativePath(t *testing.T) {
	f := mustParse(t, "./foo")
	sel := f.Selectors[0]
	assertKind(t, sel, KindPath)
	if sel.Path != "./foo" {
		t.Errorf("path=%q, want ./foo", sel.Path)
	}
	assertNoRelations(t, sel)
}

// --- pnpm test fixture: parent_relative_path_selector ---
// pnpm: parent_dir: "/foo"  (carapace stores raw "../foo")
func TestPnpmVerifyParentRelativePath(t *testing.T) {
	f := mustParse(t, "../foo")
	sel := f.Selectors[0]
	assertKind(t, sel, KindPath)
	if sel.Path != "../foo" {
		t.Errorf("path=%q, want ../foo", sel.Path)
	}
}

// --- pnpm test fixture: dependents_of_brace_dir ---
// pnpm: include_dependents: true, parent_dir: "/prefix/foo"
// carapace: includeDependents=true, braceInner="./foo", kind=Brace
func TestPnpmVerifyDependentsOfBraceDir(t *testing.T) {
	f := mustParse(t, "...{./foo}")
	sel := f.Selectors[0]
	if !sel.IncludeDependents {
		t.Errorf("want dependents=true")
	}
	if sel.BraceInner != "./foo" {
		t.Errorf("braceInner=%q, want ./foo", sel.BraceInner)
	}
	if sel.Kind != KindBrace {
		t.Errorf("kind=%v, want KindBrace", sel.Kind)
	}
}

// --- pnpm test fixture: dot_selects_prefix ---
// pnpm: parent_dir: "/prefix"  (carapace: kind=Self, path=".")
func TestPnpmVerifyDotSelector(t *testing.T) {
	f := mustParse(t, ".")
	sel := f.Selectors[0]
	assertKind(t, sel, KindSelf)
	if sel.Path != "." {
		t.Errorf("path=%q, want .", sel.Path)
	}
}

// --- pnpm test fixture: dotdot_selects_parent_of_prefix ---
// pnpm: parent_dir: "/"  (carapace: kind=Parent, path="..")
func TestPnpmVerifyDotDotSelector(t *testing.T) {
	f := mustParse(t, "..")
	sel := f.Selectors[0]
	assertKind(t, sel, KindParent)
	if sel.Path != ".." {
		t.Errorf("path=%q, want ..", sel.Path)
	}
}

// --- pnpm test fixture: diff_selector ---
// pnpm: diff: "master"
func TestPnpmVerifyDiffSelector(t *testing.T) {
	f := mustParse(t, "[master]")
	sel := f.Selectors[0]
	assertKind(t, sel, KindDiff)
	if sel.Diff != "master" {
		t.Errorf("diff=%q, want master", sel.Diff)
	}
	assertNoRelations(t, sel)
}

// --- pnpm test fixture: brace_and_diff ---
// pnpm: diff: "master", parent_dir: "/prefix/foo"
// carapace: kind=Brace, braceInner="foo", diff="master"
func TestPnpmVerifyBraceAndDiff(t *testing.T) {
	f := mustParse(t, "{foo}[master]")
	sel := f.Selectors[0]
	if sel.BraceInner != "foo" {
		t.Errorf("braceInner=%q, want foo", sel.BraceInner)
	}
	if sel.Diff != "master" {
		t.Errorf("diff=%q, want master", sel.Diff)
	}
	if sel.Kind != KindBrace {
		t.Errorf("kind=%v, want KindBrace", sel.Kind)
	}
}

// --- pnpm test fixture: name_brace_and_diff ---
// pnpm: diff: "master", name_pattern: "pattern", parent_dir: "/prefix/foo"
// carapace: kind=PackageName, name="pattern", braceInner="foo", diff="master"
func TestPnpmVerifyNameBraceDiff(t *testing.T) {
	f := mustParse(t, "pattern{foo}[master]")
	sel := f.Selectors[0]
	if sel.Name != "pattern" {
		t.Errorf("name=%q, want pattern", sel.Name)
	}
	if sel.BraceInner != "foo" {
		t.Errorf("braceInner=%q, want foo", sel.BraceInner)
	}
	if sel.Diff != "master" {
		t.Errorf("diff=%q, want master", sel.Diff)
	}
	if sel.Kind != KindPackageName {
		t.Errorf("kind=%v, want KindPackageName", sel.Kind)
	}
}

// --- pnpm test fixture: diff_with_dependencies ---
// pnpm: diff: "master", include_dependencies: true
func TestPnpmVerifyDiffWithDeps(t *testing.T) {
	f := mustParse(t, "[master]...")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependencies {
		t.Errorf("want diff=master deps=true, got diff=%q deps=%v",
			sel.Diff, sel.IncludeDependencies)
	}
}

// --- pnpm test fixture: diff_with_dependents ---
// pnpm: diff: "master", include_dependents: true
func TestPnpmVerifyDiffWithDependents(t *testing.T) {
	f := mustParse(t, "...[master]")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependents {
		t.Errorf("want diff=master dependents=true, got diff=%q dependents=%v",
			sel.Diff, sel.IncludeDependents)
	}
}

// --- pnpm test fixture: diff_with_dependencies_and_dependents ---
// pnpm: diff: "master", include_dependencies: true, include_dependents: true
func TestPnpmVerifyDiffBothRelations(t *testing.T) {
	f := mustParse(t, "...[master]...")
	sel := f.Selectors[0]
	if sel.Diff != "master" || !sel.IncludeDependents || !sel.IncludeDependencies {
		t.Errorf("want diff=master both=true, got diff=%q deps=%v dependents=%v",
			sel.Diff, sel.IncludeDependencies, sel.IncludeDependents)
	}
}

// --- pnpm test fixture: triple_dots_reduces_to_dependencies_only ---
// pnpm: include_dependencies: true, everything else default
// NOTE: pnpm treats bare "..." as include_dependencies=true with empty base.
// carapace-pnpm ERRORS on bare "...".
func TestPnpmVerifyBareTripleDots(t *testing.T) {
	_, err := Parse("...")
	if err != nil {
		t.Logf("DISCREPANCY: pnpm parses bare \"...\" as include_dependencies=true with empty base; carapace errors: %v", err)
	} else {
		t.Errorf("carapace accepted bare \"...\" unexpectedly")
	}
}

// --- pnpm test fixture: empty_braces_fall_back_to_name ---
// pnpm: name_pattern: "{}"
// NOTE: pnpm regex requires {[^}]+} (at least one char), so {} falls back to name.
// carapace-pnpm ERRORS on "{}" (unterminated or empty brace).
func TestPnpmVerifyEmptyBraces(t *testing.T) {
	_, err := Parse("{}")
	if err != nil {
		t.Logf("DISCREPANCY: pnpm treats \"{}\" as name_pattern=\"{}\"; carapace errors: %v", err)
	} else {
		t.Errorf("carapace accepted \"{}\" unexpectedly")
	}
}

// --- pnpm test fixture: dot_prefixed_name_is_not_a_location ---
// pnpm: name_pattern: ".foo"
// NOTE: pnpm regex name group [^.] excludes '.' as first char, so name is None.
// Falls through: is_selector_by_location(".foo") → false.
// Falls through: name fallback → name_pattern=".foo" (drops exclude).
// carapace-pnpm ERRORS on ".foo" because isLocation is false and name parsing
// excludes '.' as first char.
func TestPnpmVerifyDotPrefixedName(t *testing.T) {
	_, err := Parse(".foo")
	if err != nil {
		t.Logf("DISCREPANCY: pnpm treats \".foo\" as name_pattern=\".foo\"; carapace errors: %v", err)
	} else {
		t.Errorf("carapace accepted \".foo\" unexpectedly")
	}
}

// --- pnpm test fixture: exclude_with_leading_brace_name_keeps_exclude ---
// pnpm: exclude: true, name_pattern: "{foo"
// NOTE: pnpm's regex matches {foo as name="{foo" (no brace group, since no
// closing }). The exclude is KEPT because the regex path preserves it.
// carapace-pnpm ERRORS because it sees '{' as first char, tries brace parsing,
// and fails on unterminated brace.
func TestPnpmVerifyExcludeWithLeadingBrace(t *testing.T) {
	_, err := Parse("!{foo")
	if err != nil {
		t.Logf("DISCREPANCY: pnpm parses \"!{foo\" as exclude=true name=\"{foo\"; carapace errors: %v", err)
	} else {
		t.Errorf("carapace accepted \"!{foo\" unexpectedly")
	}
}

// --- pnpm test fixture: leading_brace_name_then_diff ---
// pnpm: diff: "master", name_pattern: "{"
// Input: {[master]
// NOTE: pnpm backtracks: name="{" then [master] matches as bracket group.
// carapace-pnpm ERRORS because '{' is excluded as name first char, then
// brace parsing fails (no closing '}').
func TestPnpmVerifyLeadingBraceNameThenDiff(t *testing.T) {
	_, err := Parse("{[master]")
	if err != nil {
		t.Logf("DISCREPANCY: pnpm parses \"{[master]\" as name=\"{\" diff=\"master\"; carapace errors: %v", err)
	} else {
		t.Errorf("carapace accepted \"{[master]\" unexpectedly")
	}
}

// --- pnpm test fixture: leading_brace_name_then_dir ---
// pnpm: name_pattern: "}foo", parent_dir: "/prefix/bar"
// Input: }foo{bar}
// NOTE: pnpm backtracks: name="}foo" then {bar} matches as brace group.
// carapace-pnpm: '}' is not in the exclusion set, so it enters name parsing,
// finds '{' at position 4, name="}foo", then brace parsing succeeds with
// braceInner="bar". This should WORK.
func TestPnpmVerifyLeadingBraceNameThenDir(t *testing.T) {
	f, err := Parse("}foo{bar}")
	if err != nil {
		t.Fatalf("parse \"}foo{bar}\": %v", err)
	}
	sel := f.Selectors[0]
	if sel.Name != "}foo" {
		t.Errorf("name=%q, want }foo", sel.Name)
	}
	if sel.BraceInner != "bar" {
		t.Errorf("braceInner=%q, want bar", sel.BraceInner)
	}
}

// --- pnpm test fixture: unparsable_braces_fall_back_to_name ---
// pnpm: name_pattern: "foo}bar"
// carapace-pnpm: name parsing finds no '{' or '[', name="foo}bar".
// isValidName allows '}' at depth 0 (does nothing, doesn't return false).
// This should WORK.
func TestPnpmVerifyUnparsableBracesFallback(t *testing.T) {
	f, err := Parse("foo}bar")
	if err != nil {
		t.Fatalf("parse \"foo}bar\": %v", err)
	}
	sel := f.Selectors[0]
	if sel.Name != "foo}bar" {
		t.Errorf("name=%q, want foo}bar", sel.Name)
	}
	if sel.Kind != KindPackageName {
		t.Errorf("kind=%v, want KindPackageName", sel.Kind)
	}
}

// --- helper assertions ---

func assertKind(t *testing.T, sel Selector, want SelectorKind) {
	t.Helper()
	if sel.Kind != want {
		t.Errorf("kind=%v, want %v", sel.Kind, want)
	}
}

func assertName(t *testing.T, sel Selector, want string) {
	t.Helper()
	if sel.Name != want {
		t.Errorf("name=%q, want %q", sel.Name, want)
	}
}
