package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/carapace-sh/carapace"
	"github.com/carapace-sh/carapace-pnpm/pkg/actions/tools/pnpm"
	pnpmparser "github.com/carapace-sh/carapace-pnpm/pkg/pnpm"
	"github.com/spf13/cobra"
)

var filterCmd = &cobra.Command{
	Use:   "filter <selector>",
	Short: "Parse a pnpm filter selector",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := pnpmparser.Parse(args[0])
		if err != nil {
			return err
		}
		m, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(m))
		return nil
	},
}

var filterCompleteCmd = &cobra.Command{
	Use:   "filter-complete <selector>",
	Short: "Get completion context for a pnpm filter selector",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := pnpmparser.ParseForCompletion(args[0])
		m, err := json.MarshalIndent(ctx, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(m))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(filterCmd)
	rootCmd.AddCommand(filterCompleteCmd)

	carapace.Gen(filterCmd).PositionalAnyCompletion(pnpm.ActionFilters())
	carapace.Gen(filterCompleteCmd).PositionalAnyCompletion(pnpm.ActionFilters())
}
