package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/feedistribution/types"
)

// GetQueryCmd returns the query commands for the feedistribution module
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQueryRevenue(),
		GetCmdQueryRevenues(),
		GetCmdQueryDeployerRevenues(),
		GetCmdQueryWithdrawerRevenues(),
	)

	return cmd
}

// GetCmdQueryParams implements the query params command.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current feedistribution parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			res, err := queryClient.Params(context.Background(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryRevenue implements the query revenue command.
func GetCmdQueryRevenue() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revenue CONTRACT_ADDRESS",
		Short: "Query revenue for a contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Validate contract address
			if err := types.ValidateAddress(args[0]); err != nil {
				return fmt.Errorf("invalid contract address: %w", err)
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryRevenueRequest{
				ContractAddress: args[0],
			}

			res, err := queryClient.Revenue(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryRevenues implements the query revenues command.
func GetCmdQueryRevenues() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revenues",
		Short: "Query all revenues",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryRevenuesRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.Revenues(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "revenues")
	return cmd
}

// GetCmdQueryDeployerRevenues implements the query revenues by deployer command.
func GetCmdQueryDeployerRevenues() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deployer-revenues DEPLOYER_ADDRESS",
		Short: "Query all revenues by deployer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Validate deployer address
			_, err = sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid deployer address: %w", err)
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryDeployerRevenuesRequest{
				DeployerAddress: args[0],
				Pagination:      pageReq,
			}

			res, err := queryClient.DeployerRevenues(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "deployer revenues")
	return cmd
}

// GetCmdQueryWithdrawerRevenues implements the query revenues by withdrawer command.
func GetCmdQueryWithdrawerRevenues() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdrawer-revenues WITHDRAWER_ADDRESS",
		Short: "Query all revenues by withdrawer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			// Validate withdrawer address
			_, err = sdk.AccAddressFromBech32(args[0])
			if err != nil {
				return fmt.Errorf("invalid withdrawer address: %w", err)
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			req := &types.QueryWithdrawerRevenuesRequest{
				WithdrawerAddress: args[0],
				Pagination:        pageReq,
			}

			res, err := queryClient.WithdrawerRevenues(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "withdrawer revenues")
	return cmd
}
