package cli

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"

	"helios-core/helios-chain/x/logos/types"
)

func GetQueryCmd() *cobra.Command {
	hyperionQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the logos module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	hyperionQueryCmd.AddCommand([]*cobra.Command{}...)

	return hyperionQueryCmd
}
