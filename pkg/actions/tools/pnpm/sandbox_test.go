package pnpm

import (
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace/pkg/sandbox"
)

func TestActionFiltersEmpty(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("").Expect(
			carapace.Batch(
				carapace.ActionValuesDescribed(
					".", "the package in the current directory",
				).Tag("self"),
				carapace.ActionValuesDescribed(
					"./", "packages matching a path glob",
					"{./", "glob group, e.g. {./apps/*,./packages/*}",
					"[", "packages changed since a git ref, e.g. [master]",
				).NoSpace().Tag("path glob"),
			).ToA(),
		)
	})
}

func TestActionFiltersAfterBaseExpectsRelation(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("foo").Expect(
			carapace.ActionValuesDescribed(
				"...", "the package and its dependencies",
				"^...", "the package's dependencies, excluding itself",
			).NoSpace(),
		)
	})
}

func TestActionFiltersAfterCommaExpectsSelector(t *testing.T) {
	sandbox.Action(t, func() carapace.Action {
		return ActionFilters()
	})(func(s *sandbox.Sandbox) {
		s.Run("foo,").Expect(
			carapace.Batch(
				carapace.ActionValuesDescribed(
					".", "the package in the current directory",
				).Tag("self"),
				carapace.ActionValuesDescribed(
					"./", "packages matching a path glob",
					"{./", "glob group, e.g. {./apps/*,./packages/*}",
					"[", "packages changed since a git ref, e.g. [master]",
				).NoSpace().Tag("path glob"),
			).ToA().Prefix("foo,"),
		)
	})
}
