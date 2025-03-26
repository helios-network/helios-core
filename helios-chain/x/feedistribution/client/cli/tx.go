package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/feedistribution/types"
)

// GetTxCmd returns the transaction commands for the feedistribution module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		NewRegisterRevenueCmd(),
		NewUpdateRevenueCmd(),
		NewCancelRevenueCmd(),
	)

	return cmd
}

// NewRegisterRevenueCmd returns a CLI command handler for registering a
// contract for fee distribution
func NewRegisterRevenueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-revenue CONTRACT_ADDRESS [WITHDRAWER_ADDRESS]",
		Short: "Register a contract for fee distribution",
		Long: `Register a contract for fee distribution. The deployer can optionally specify a withdrawer address.
If no withdrawer address is specified, the deployer address will be used as the withdrawer.

Example:
$ heliades tx feedistribution register-revenue 0x... --from mykey
$ heliades tx feedistribution register-revenue 0x... 0x... --from mykey
`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get deployer address from the --from flag
			deployer := clientCtx.GetFromAddress()

			// Validate contract address
			if err := types.ValidateAddress(args[0]); err != nil {
				return fmt.Errorf("invalid contract address: %w", err)
			}

			// Parse withdrawer address if provided, otherwise use deployer
			var withdrawer sdk.AccAddress
			if len(args) == 2 {
				withdrawer, err = sdk.AccAddressFromBech32(args[1])
				if err != nil {
					return fmt.Errorf("invalid withdrawer address: %w", err)
				}
			} else {
				withdrawer = deployer
			}

			msg := &types.MsgRegisterRevenue{
				ContractAddress:   args[0],
				DeployerAddress:   deployer.String(),
				WithdrawerAddress: withdrawer.String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewUpdateRevenueCmd returns a CLI command handler for updating the withdrawer
// address for a registered contract
func NewUpdateRevenueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-revenue CONTRACT_ADDRESS WITHDRAWER_ADDRESS",
		Short: "Update the withdrawer address for a registered contract",
		Long: `Update the withdrawer address for a registered contract. Only the deployer can update the withdrawer.

Example:
$ heliades tx feedistribution update-revenue 0x... cosmos1... --from mykey
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get deployer address from the --from flag
			deployer := clientCtx.GetFromAddress()

			// Validate contract address
			if err := types.ValidateAddress(args[0]); err != nil {
				return fmt.Errorf("invalid contract address: %w", err)
			}

			// Parse withdrawer address
			withdrawer, err := sdk.AccAddressFromBech32(args[1])
			if err != nil {
				return fmt.Errorf("invalid withdrawer address: %w", err)
			}

			msg := &types.MsgUpdateRevenue{
				ContractAddress:   args[0],
				DeployerAddress:   deployer.String(),
				WithdrawerAddress: withdrawer.String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

// NewCancelRevenueCmd returns a CLI command handler for removing a contract
// from fee distribution
func NewCancelRevenueCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-revenue CONTRACT_ADDRESS",
		Short: "Remove a contract from fee distribution",
		Long: `Remove a contract from fee distribution. Only the deployer can cancel the revenue.

Example:
$ heliades tx feedistribution cancel-revenue 0x... --from mykey
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			// Get deployer address from the --from flag
			deployer := clientCtx.GetFromAddress()

			// Validate contract address
			if err := types.ValidateAddress(args[0]); err != nil {
				return fmt.Errorf("invalid contract address: %w", err)
			}

			msg := &types.MsgCancelRevenue{
				ContractAddress: args[0],
				DeployerAddress: deployer.String(),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}
