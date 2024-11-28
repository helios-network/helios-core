package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"

	"helios-core/helios-chain/modules/peggy/types"
)

func CmdGetAttestation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attestation [nonce] [claim-hash]",
		Short: "Query an attestation by nonce and claim hash",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			nonce, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			claimHash, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAttestationRequest{
				Nonce:     nonce,
				ClaimHash: claimHash,
			}

			res, err := queryClient.Attestation(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
