package keeper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/modules/hyperion/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	erc20types "helios-core/helios-chain/x/erc20/types"

	"github.com/Helios-Chain-Labs/metrics"
)

// AttestationHandler processes `observed` Attestations
type AttestationHandler struct {
	keeper      Keeper
	bankKeeper  types.BankKeeper
	erc20Keeper types.ERC20Keeper
	svcTags     metrics.Tags
}

func NewAttestationHandler(bankKeeper types.BankKeeper, keeper Keeper) AttestationHandler {
	return AttestationHandler{
		keeper:     keeper,
		bankKeeper: bankKeeper,

		svcTags: metrics.Tags{
			"svc": "hyperion_att",
		},
	}
}

// Handle is the entry point for Attestation processing.
func (a AttestationHandler) Handle(ctx sdk.Context, claim types.EthereumClaim) error {
	metrics.ReportFuncCall(a.svcTags)
	doneFn := metrics.ReportFuncTiming(a.svcTags)
	defer doneFn()

	fmt.Println("Handle======================= claim type", claim.GetType())
	switch claim := claim.(type) {
	// deposit in this context means a deposit into the Ethereum side of the bridge
	case *types.MsgDepositClaim:
		invalidAddress := false

		ethereumSender, errEthereumSender := types.NewEthAddress(claim.EthereumSender)
		// likewise nil sender would have to be caused by a bogus event
		if errEthereumSender != nil {
			metrics.ReportFuncError(a.svcTags)
			return errors.Wrap(types.ErrInvalidEthSender, "invalid ethereum sender on claim")
		}

		// Check if coin is Cosmos-originated asset and get denom
		isCosmosOriginated, denom := a.keeper.ERC20ToDenomLookup(ctx, common.HexToAddress(claim.TokenContract), claim.HyperionId)

		coins := sdk.Coins{
			sdk.NewCoin(denom, claim.Amount),
		}

		addr, err := sdk.AccAddressFromBech32(claim.CosmosReceiver)
		if err != nil {
			metrics.ReportFuncError(a.svcTags)
			invalidAddress = true
		}

		// Block blacklisted asset transfers
		// (these funds are unrecoverable for the blacklisted sender, they will instead be sent to community pool)
		if a.keeper.IsOnBlacklist(ctx, *ethereumSender) {
			metrics.ReportFuncError(a.svcTags)
			invalidAddress = true
		}
		fmt.Println("isCosmosOriginated", isCosmosOriginated)

		if !isCosmosOriginated {
			// Check if supply overflows with claim amount
			currentSupply := a.bankKeeper.GetSupply(ctx, denom)
			fmt.Println("currentSupply", currentSupply)
			newSupply := new(big.Int).Add(currentSupply.Amount.BigInt(), claim.Amount.BigInt())
			fmt.Println("newSupply", newSupply)
			if newSupply.BitLen() > 256 {
				metrics.ReportFuncError(a.svcTags)
				return errors.Wrap(types.ErrSupplyOverflow, "invalid supply")
			}

			if err := a.bankKeeper.MintCoins(ctx, types.ModuleName, coins); err != nil {
				metrics.ReportFuncError(a.svcTags)
				return errors.Wrapf(err, "mint vouchers coins: %s", coins)
			}
		}

		if !invalidAddress { // address appears valid, attempt to send minted/locked coins to receiver
			if err = a.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
				metrics.ReportFuncError(a.svcTags)
				invalidAddress = true
			}
			pairID := a.erc20Keeper.GetTokenPairID(ctx, denom)
			pair, found := a.erc20Keeper.GetTokenPair(ctx, pairID)
			if !found {
				// Check if metadata already exists for this base denom
				denomMetadata, _ := a.bankKeeper.GetDenomMetaData(ctx, denom)
				symbol := denomMetadata.Symbol
				decimals := denomMetadata.Decimals

				coinMetadata := banktypes.Metadata{
					Description: fmt.Sprintf("Token %s created with Hyperion", denom),
					Base:        denom,
					Symbol:      symbol,
					Decimals:    uint32(decimals),
					DenomUnits: []*banktypes.DenomUnit{
						{
							Denom:    denom,
							Exponent: 0,
						},
						{
							Denom:    denom,
							Exponent: uint32(decimals),
						},
					},
				}
				erc20Contract, err := a.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
				if err != nil {
					metrics.ReportFuncError(a.svcTags)
					return errors.Wrap(err, "failed to deploy ERC20 contract")
				}
				tokenPair := erc20types.NewTokenPair(erc20Contract, denom, erc20types.OWNER_MODULE)
				pair = tokenPair
				a.erc20Keeper.SetToken(ctx, tokenPair)
			
				// Enable dynamic precompiles for the deployed ERC20 contract
				a.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
			}
			if err := a.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, coins); err != nil {
				metrics.ReportFuncError(a.svcTags)
				return errors.Wrap(err, "failed to send coins from account to module")
			}
			// mint ERC20 tokens
			if err := a.erc20Keeper.MintERC20Tokens(ctx, pair.GetERC20Contract(), common.BytesToAddress(addr), claim.Amount.BigInt()); err != nil {
				metrics.ReportFuncError(a.svcTags)
				return errors.Wrap(err, "failed to mint ERC20 tokens")
			}
		}

		// for whatever reason above, blacklisted, invalid string, etc this deposit is not valid
		// we can't send the tokens back on the Ethereum side, and if we don't put them somewhere on
		// the cosmos side they will be lost an inaccessible even though they are locked in the bridge.
		// so we deposit the tokens into the community pool for later use via governance vote
		if invalidAddress {
			if err := a.keeper.SendToCommunityPool(ctx, coins); err != nil {
				return errors.Wrap(err, "failed to send to Community pool")
			}
		}

		// withdraw in this context means a withdraw from the Ethereum side of the bridge
	case *types.MsgWithdrawClaim:
		tokenContract := common.HexToAddress(claim.TokenContract)
		a.keeper.OutgoingTxBatchExecuted(ctx, tokenContract, claim.BatchNonce, claim.HyperionId)
		return nil
	case *types.MsgERC20DeployedClaim:
		// Check if it already exists
		existingERC20, exists := a.keeper.GetCosmosOriginatedERC20ByHyperionID(ctx, claim.CosmosDenom, claim.HyperionId)
		if exists {
			metrics.ReportFuncError(a.svcTags)

			return errors.Wrap(
				types.ErrInvalid,
				fmt.Sprintf("ERC20 %s already exists for denom %s", existingERC20, claim.CosmosDenom))
		}

		// Check if denom exists
		metadata, found := a.keeper.bankKeeper.GetDenomMetaData(ctx, claim.CosmosDenom)
		if metadata.Base == "" || !found {
			metrics.ReportFuncError(a.svcTags)
			return errors.Wrap(types.ErrUnknown, fmt.Sprintf("denom not found %s", claim.CosmosDenom))
		}

		// Check if attributes of ERC20 match Cosmos denom
		if claim.Name != metadata.Display {
			metrics.ReportFuncError(a.svcTags)
			return errors.Wrap(
				types.ErrInvalid,
				fmt.Sprintf("ERC20 name %s does not match denom display %s", claim.Name, metadata.Description))
		}

		if claim.Symbol != metadata.Display {
			metrics.ReportFuncError(a.svcTags)
			return errors.Wrap(
				types.ErrInvalid,
				fmt.Sprintf("ERC20 symbol %s does not match denom display %s", claim.Symbol, metadata.Display))
		}

		// ERC20 tokens use a very simple mechanism to tell you where to display the decimal point.
		// The "decimals" field simply tells you how many decimal places there will be.
		// Cosmos denoms have a system that is much more full featured, with enterprise-ready token denominations.
		// There is a DenomUnits array that tells you what the name of each denomination of the
		// token is.
		// To correlate this with an ERC20 "decimals" field, we have to search through the DenomUnits array
		// to find the DenomUnit which matches up to the main token "display" value. Then we take the
		// "exponent" from this DenomUnit.
		// If the correct DenomUnit is not found, it will default to 0. This will result in there being no decimal places
		// in the token's ERC20 on Ethereum. So, for example, if this happened with Atom, 1 Atom would appear on Ethereum
		// as 1 million Atoms, having 6 extra places before the decimal point.
		// This will only happen with a Denom Metadata which is for all intents and purposes invalid, but I am not sure
		// this is checked for at any other point.
		decimals := uint32(0)
		for _, denomUnit := range metadata.DenomUnits {
			if denomUnit.Denom == metadata.Display {
				decimals = denomUnit.Exponent
				break
			}
		}

		if uint64(decimals) != claim.Decimals {
			metrics.ReportFuncError(a.svcTags)
			return errors.Wrap(
				types.ErrInvalid,
				fmt.Sprintf("ERC20 decimals %d does not match denom decimals %d", claim.Decimals, decimals))
		}

		// Add to denom-erc20 mapping
		a.keeper.SetCosmosOriginatedDenomToERC20ByHyperionID(ctx, claim.CosmosDenom, common.HexToAddress(claim.TokenContract), claim.HyperionId)
	case *types.MsgValsetUpdatedClaim:
		// TODO here we should check the contents of the validator set against
		// the store, if they differ we should take some action to indicate to the
		// user that bridge highjacking has occurred
		a.keeper.SetLastObservedValset(ctx, types.Valset{
			Nonce:        claim.ValsetNonce,
			Members:      claim.Members,
			RewardAmount: claim.RewardAmount,
			RewardToken:  claim.RewardToken,
		})
	default:
		metrics.ReportFuncError(a.svcTags)
		panic(fmt.Sprintf("Invalid event type for attestations %s", claim.GetType()))
	}

	return nil
}
