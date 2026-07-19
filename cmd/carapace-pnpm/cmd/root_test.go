package cmd

import (
	"testing"

	"github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
)

var parseSuccessCases = []string{
	// Names
	"foo",
	"@scope/bar",
	// Locations
	".",
	"..",
	"./packages/*",
	"../shared",
	// Relational modifiers (all combinations)
	"foo...",
	"...foo",
	"...foo...",
	"foo^...",
	"...^foo",
	"...^foo^...",
	// Negation
	"!foo",
	"!foo...",
	// Diff selectors
	"[master]",
	"[master]...",
	"...[master]",
	"...[master]...",
	// Brace selectors
	"{foo}",
	"{./apps/*,./packages/*}",
	"...{./foo}",
	// Combinations
	"pattern{foo}[master]",
	"{foo}[master]",
	// Multiple selectors
	"foo,bar",
	"./packages/app...,./packages/widget",
	"foo..., !bar",
}

func TestParseAllSuccessCases(t *testing.T) {
	for _, sel := range parseSuccessCases {
		_, err := pnpm.Parse(sel)
		if err != nil {
			t.Errorf("parse %q: %v", sel, err)
		}
	}
}

var completionTestCases = []string{
	"",
	"foo",
	"foo...",
	"foo^...",
	"...foo",
	"...foo...",
	"...^foo",
	"!",
	"!foo",
	"foo,",
	"foo, ",
	"./packages/*",
	"@scope/bar",
	".",
	"..",
	"foo,bar",
	"[master]",
	"{foo}",
}

func TestCompletionAllCases(t *testing.T) {
	for _, input := range completionTestCases {
		ctx := pnpm.ParseForCompletion(input)
		if len(ctx.ExpectedTokens) == 0 {
			t.Errorf("completion %q: no expected tokens", input)
		}
	}
}
