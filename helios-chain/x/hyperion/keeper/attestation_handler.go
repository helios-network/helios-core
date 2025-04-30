package keeper

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"helios-core/helios-chain/x/hyperion/types"

	"github.com/Helios-Chain-Labs/metrics"

	erc20types "helios-core/helios-chain/x/erc20/types"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// AttestationHandler processes `observed` Attestations
type AttestationHandler struct {
	keeper     Keeper
	bankKeeper types.BankKeeper
	svcTags    metrics.Tags
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
		tokenAddressToDenom, exists := a.keeper.GetTokenFromAddress(ctx, claim.HyperionId, common.HexToAddress(claim.TokenContract))
		if !exists {
			hyperionDenom := types.NewHyperionDenom(claim.HyperionId, common.HexToAddress(claim.TokenContract))
			tokenAddressToDenom = a.keeper.SetToken(ctx, claim.HyperionId, &types.TokenAddressToDenom{
				TokenAddress:       common.HexToAddress(claim.TokenContract).String(),
				Denom:              hyperionDenom.String(),
				IsCosmosOriginated: false,
			})
		}
		denom := tokenAddressToDenom.Denom

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
		fmt.Println("isCosmosOriginated", tokenAddressToDenom.IsCosmosOriginated)

		if !tokenAddressToDenom.IsCosmosOriginated {

			name := denom
			symbol := denom
			decimals := uint64(18)

			if claim.Data != "" {
				metadata, _, _ := a.keeper.parseClaimData(ctx, claim.Data)
				if metadata != nil {
					name = metadata.Name
					symbol = metadata.Symbol
					decimals = metadata.Decimals

					if symbol == "" {
						symbol = denom
					}
				}
			}

			_, ok := a.keeper.erc20Keeper.GetTokenPair(ctx, a.keeper.erc20Keeper.GetTokenPairID(ctx, denom))

			// if pair doens't exists creation of the erc20 token and link it to denom
			if !ok {

				coinMetadata := banktypes.Metadata{
					Description: fmt.Sprintf("Token %s created with Hyperion", denom),
					Base:        denom,
					Name:        name,
					Symbol:      symbol,
					Decimals:    uint32(decimals),
					Display:     name,
					DenomUnits: []*banktypes.DenomUnit{
						{
							Denom:    denom,
							Exponent: 0,
						},
						{
							Denom:    symbol,
							Exponent: uint32(decimals),
						},
					},
					ChainsMetadatas: []*banktypes.ChainMetadata{
						{
							ChainId:         a.keeper.GetChainIdFromHyperionId(ctx, claim.HyperionId),
							ContractAddress: common.HexToAddress(claim.TokenContract).String(),
							Symbol:          symbol,
							Decimals:        uint32(decimals),
							IsOriginated:    true,
						},
					},
				}

				if err := coinMetadata.Validate(); err != nil {
					return fmt.Errorf("invalid coin metadata: %w", err)
				}
				////////////////////////////////////////////////////////
				// Generate logo
				// base64Logo, err := rpctypes.GenerateTokenLogoBase64(symbol)
				// if err == nil {
				// 	logoHash, err := a.keeper.logosKeeper.StoreLogo(ctx, base64Logo)
				// 	if err == nil {
				// 		coinMetadata.Logo = logoHash
				// 	}
				// }
				////////////////////////////////////////////////////////
				contractAddr, err := a.keeper.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
				if err != nil {
					return fmt.Errorf("failed to deploy ERC20 contract: %w", err)
				}
				tokenPair := erc20types.NewTokenPair(contractAddr, denom, erc20types.OWNER_MODULE)
				a.keeper.erc20Keeper.SetToken(ctx, tokenPair)
				a.keeper.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())
			}

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
		a.keeper.UpdateRpcUsed(ctx, claim.HyperionId, claim.RpcUsed, claim.BlockHeight)
		// withdraw in this context means a withdraw from the Ethereum side of the bridge
	case *types.MsgWithdrawClaim:
		tokenContract := common.HexToAddress(claim.TokenContract)
		a.keeper.OutgoingTxBatchExecuted(ctx, tokenContract, claim.BatchNonce, claim.HyperionId, claim)
		a.keeper.UpdateRpcUsed(ctx, claim.HyperionId, claim.RpcUsed, claim.BlockHeight)
		return nil
	case *types.MsgERC20DeployedClaim:
		// Check if it already exists
		tokenAddressToDenom, exists := a.keeper.GetTokenFromDenom(ctx, claim.HyperionId, claim.CosmosDenom)
		if exists {
			metrics.ReportFuncError(a.svcTags)

			return errors.Wrap(
				types.ErrInvalid,
				fmt.Sprintf("ERC20 %s already exists for denom %s", tokenAddressToDenom, claim.CosmosDenom))
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

		for _, chainMetadata := range metadata.ChainsMetadatas {
			if chainMetadata.ChainId == a.keeper.GetChainIdFromHyperionId(ctx, claim.HyperionId) {
				metrics.ReportFuncError(a.svcTags)
				return errors.Wrap(
					types.ErrInvalid,
					fmt.Sprintf("ERC20 %s already exists for chain %d", claim.CosmosDenom, chainMetadata.ChainId))
			}
		}

		// Token addresses use a very simple mechanism to tell you where to display the decimal point.
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

		// Add to denom-token address mapping
		a.keeper.SetToken(ctx, claim.HyperionId, &types.TokenAddressToDenom{
			TokenAddress:       common.HexToAddress(claim.TokenContract).String(),
			Denom:              claim.CosmosDenom,
			IsCosmosOriginated: true,
		})
		metadata.ChainsMetadatas = append(metadata.ChainsMetadatas, &banktypes.ChainMetadata{
			ChainId:         a.keeper.GetChainIdFromHyperionId(ctx, claim.HyperionId),
			ContractAddress: common.HexToAddress(claim.TokenContract).String(),
			Symbol:          claim.Symbol,
			Decimals:        uint32(decimals),
			IsOriginated:    false,
		})
		a.keeper.bankKeeper.SetDenomMetaData(ctx, metadata)
		a.keeper.UpdateRpcUsed(ctx, claim.HyperionId, claim.RpcUsed, claim.BlockHeight)
	case *types.MsgValsetUpdatedClaim:
		// TODO here we should check the contents of the validator set against
		// the store, if they differ we should take some action to indicate to the
		// user that bridge highjacking has occurred
		a.keeper.SetLastObservedValset(ctx, claim.HyperionId, types.Valset{
			Nonce:        claim.ValsetNonce,
			Members:      claim.Members,
			RewardAmount: claim.RewardAmount,
			RewardToken:  claim.RewardToken,
		})
		a.keeper.UpdateRpcUsed(ctx, claim.HyperionId, claim.RpcUsed, claim.BlockHeight)
	default:
		metrics.ReportFuncError(a.svcTags)
		panic(fmt.Sprintf("Invalid event type for attestations %s", claim.GetType()))
	}

	return nil
}
