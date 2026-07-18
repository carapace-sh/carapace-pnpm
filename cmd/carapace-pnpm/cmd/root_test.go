package cmd

import (
	"testing"

	"github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
)

var parseSuccessCases = []string{
	"foo",
	"@scope/bar",
	".",
	"./packages/*",
	"foo...",
	"foo^...",
	"!foo",
	"foo,bar",
	"./packages/app...,./packages/widget",
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
	"!",
	"!foo",
	"foo,",
	"foo, ",
	"./packages/*",
	"@scope/bar",
	".",
	"foo,bar",
}

func TestCompletionAllCases(t *testing.T) {
	for _, input := range completionTestCases {
		ctx := pnpm.ParseForCompletion(input)
		if len(ctx.ExpectedTokens) == 0 {
			t.Errorf("completion %q: no expected tokens", input)
		}
	}
}
