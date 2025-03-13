//nolint:staticcheck // deprecated gov proposal flags
package cli

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cliflags "helios-core/cli/flags"
	"helios-core/helios-chain/modules/hyperion/types"
)

func GetTxCmd(storeKey string) *cobra.Command {
	hyperionTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Hyperion transaction subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	hyperionTxCmd.AddCommand([]*cobra.Command{
		CmdSendToChain(),
		CmdRequestBatch(),
		CmdSetOrchestratorAddress(),
		GetUnsafeTestingCmd(),
		NewCancelSendToChain(),
		BlacklistEthereumAddresses(),
		RevokeBlacklistEthereumAddresses(),
	}...)

	return hyperionTxCmd
}

func GetUnsafeTestingCmd() *cobra.Command {
	testingTxCmd := &cobra.Command{
		Use:                        "unsafe_testing",
		Short:                      "helpers for testing. not going into production",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	testingTxCmd.AddCommand([]*cobra.Command{
		CmdUnsafeETHPrivKey(),
		CmdUnsafeETHAddr(),
	}...)

	return testingTxCmd
}

func CmdUnsafeETHPrivKey() *cobra.Command {
	return &cobra.Command{
		Use:   "gen-eth-key",
		Short: "Generate and print a new ecdsa key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := ethCrypto.GenerateKey()
			if err != nil {
				return errors.Wrap(err, "can not generate key")
			}
			k := "0x" + hex.EncodeToString(ethCrypto.FromECDSA(key))
			println(k)
			return nil
		},
	}
}

func CmdUnsafeETHAddr() *cobra.Command {
	return &cobra.Command{
		Use:   "eth-address",
		Short: "Print address for an ECDSA eth key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			privKeyString := args[0][2:]
			privateKey, err := ethCrypto.HexToECDSA(privKeyString)
			if err != nil {
				log.Fatal(err)
			}
			// You've got to do all this to get an Eth address from the private key
			publicKey := privateKey.Public()
			publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
			if !ok {
				log.Fatal("error casting public key to ECDSA")
			}
			ethAddress := ethCrypto.PubkeyToAddress(*publicKeyECDSA).Hex()
			println(ethAddress)
			return nil
		},
	}
}

func CmdSendToChain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send-to-chain [dest-hyperion-id] [eth-dest] [amount] [bridge-fee]",
		Short: "Adds a new entry to the transaction pool to withdraw an amount from the Ethereum bridge contract",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()
			destHypperionId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errors.Wrap(err, "dest hypperionId")
			}
			
			amount, err := sdk.ParseCoinsNormalized(args[2])
			if err != nil {
				return errors.Wrap(err, "amount")
			}
			bridgeFee, err := sdk.ParseCoinsNormalized(args[3])
			if err != nil {
				return errors.Wrap(err, "bridge fee")
			}

			if len(amount) > 1 || len(bridgeFee) > 1 {
				return fmt.Errorf("coin amounts too long, expecting just 1 coin amount for both amount and bridgeFee")
			}

			// Make the message
			msg := types.MsgSendToChain{
				Sender:         cosmosAddr.String(),
				DestHyperionId: destHypperionId,
				Dest:           args[1],
				Amount:         amount[0],
				BridgeFee:      bridgeFee[0],
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func NewCancelSendToChain() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cancel-send-to-chain [id]",
		Short: "Cancels send to chain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()

			id, _ := strconv.Atoi(args[0])
			// Make the message
			msg := types.MsgCancelSendToChain{
				TransactionId: uint64(id),
				Sender:        cosmosAddr.String(),
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdRequestBatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build-batch [dest-hyperion-id] [denom]",
		Short: "Build a new batch on the cosmos side for pooled withdrawal transactions",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			cosmosAddr := cliCtx.GetFromAddress()

			destHypperionId, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return errors.Wrap(err, "dest hypperionId")
			}
			denom := args[1]

			msg := types.MsgRequestBatch{
				HyperionId:   destHypperionId,
				Orchestrator: cosmosAddr.String(),
				Denom:        denom,
			}

			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdSetOrchestratorAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-orchestrator-address [validator-acc-address] [orchestrator-acc-address] [ethereum-address]",
		Short: "Allows validators to delegate their voting responsibilities to a given key.",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			hyperionId, err := strconv.ParseUint(args[3], 10, 64)
			if err != nil {
				return errors.Wrap(err, "hyperionId")
			}
			msg := types.MsgSetOrchestratorAddresses{
				Sender:       args[0],
				Orchestrator: args[1],
				EthAddress:   args[2],
				HyperionId:   hyperionId,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func BlacklistEthereumAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blacklist-ethereum-addresses [addresses]",
		Short: "Blacklist Ethereum addresses",
		Long: `"Example:
		heliades tx hyperion blacklist-ethereum-addresses "0x0000000000000000000000000000000000000000,0x1111111111111111111111111111111111111111"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			addressList := strings.Split(args[0], ",")
			msg := types.MsgBlacklistEthereumAddresses{
				Signer:             cliCtx.GetFromAddress().String(),
				BlacklistAddresses: addressList,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}

func RevokeBlacklistEthereumAddresses() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke-blacklist-ethereum-addresses [addresses]",
		Short: "Revoke Blacklist Ethereum addresses",
		Long: `"Example:
		heliades tx hyperion revoke-blacklist-ethereum-addresses "0x0000000000000000000000000000000000000000,0x1111111111111111111111111111111111111111"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cliCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			addressList := strings.Split(args[0], ",")
			msg := types.MsgRevokeEthereumBlacklist{
				Signer:             cliCtx.GetFromAddress().String(),
				BlacklistAddresses: addressList,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			// Send it
			return tx.GenerateOrBroadcastTxCLI(cliCtx, cmd.Flags(), &msg)
		},
	}
	cliflags.AddTxFlagsToCmd(cmd)
	return cmd
}
