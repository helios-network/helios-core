package keeper

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	erc20types "helios-core/helios-chain/x/erc20/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"

	"github.com/Helios-Chain-Labs/metrics"

	cmn "helios-core/helios-chain/precompiles/common"
	"helios-core/helios-chain/x/hyperion/types"
)

func (k *Keeper) GetTokenFromAddress(ctx sdk.Context, hyperionId uint64, tokenContract common.Address) (*types.TokenAddressToDenom, bool) {
	chainId := k.GetChainIdFromHyperionId(ctx, hyperionId)
	denom, exists := k.bankKeeper.GetDenomFromChainIdAndContractAddress(ctx, chainId, tokenContract.Hex())
	if !exists {
		return nil, false
	}

	metadata, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		return nil, false
	}

	isCosmosOriginated := true

	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.IsOriginated {
			isCosmosOriginated = false
			break
		}
	}

	isConcensusToken := k.erc20Keeper.IsAssetWhitelisted(ctx, denom)

	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.ChainId == chainId {
			return &types.TokenAddressToDenom{
				ChainId:            strconv.FormatUint(chainId, 10),
				TokenAddress:       chainMetadata.ContractAddress,
				Symbol:             chainMetadata.Symbol,
				Decimals:           uint64(chainMetadata.Decimals),
				Denom:              denom,
				IsCosmosOriginated: isCosmosOriginated,
				IsConcensusToken:   isConcensusToken,
			}, true
		}
	}

	// If the token is not found in the metadata, return nil
	return nil, false
}

func (k *Keeper) GetTokenFromDenom(ctx sdk.Context, hyperionId uint64, denom string) (*types.TokenAddressToDenom, bool) {
	chainId := k.GetChainIdFromHyperionId(ctx, hyperionId)
	metadata, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		return nil, false
	}

	isCosmosOriginated := true

	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.IsOriginated {
			isCosmosOriginated = false
			break
		}
	}

	isConcensusToken := k.erc20Keeper.IsAssetWhitelisted(ctx, denom)

	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.ChainId == chainId {
			return &types.TokenAddressToDenom{
				ChainId:            strconv.FormatUint(chainId, 10),
				TokenAddress:       chainMetadata.ContractAddress,
				Symbol:             chainMetadata.Symbol,
				Decimals:           uint64(chainMetadata.Decimals),
				Denom:              denom,
				IsCosmosOriginated: isCosmosOriginated,
				IsConcensusToken:   isConcensusToken,
			}, true
		}
	}

	// If the token is not found in the metadata, return nil
	return nil, false
}

func (k *Keeper) RemoveTokenFromChainMetadata(ctx sdk.Context, hyperionId uint64, denom string) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	metadata, found := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !found {
		return
	}
	// remove the token from the metadata
	for i, chainM := range metadata.ChainsMetadatas {
		if chainM.ChainId == k.GetChainIdFromHyperionId(ctx, hyperionId) {
			metadata.ChainsMetadatas = append(metadata.ChainsMetadatas[:i], metadata.ChainsMetadatas[i+1:]...)
			break
		}
	}
	k.bankKeeper.SetDenomMetaData(ctx, metadata)
}

func (k *Keeper) SetTokenToChainMetadata(ctx sdk.Context, chainId uint64, token *types.TokenAddressToDenom) *types.TokenAddressToDenom {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	metadata, _ := k.bankKeeper.GetDenomMetaData(ctx, token.Denom)
	chainMetadata := &banktypes.ChainMetadata{
		ChainId:         chainId,
		ContractAddress: common.HexToAddress(token.TokenAddress).String(),
		Symbol:          metadata.Symbol,
		Decimals:        uint32(metadata.Decimals),
		IsOriginated:    !token.IsCosmosOriginated,
		TotalSupply:     nil,
	}

	if !chainMetadata.IsOriginated {
		totalSupply := math.NewInt(0)
		chainMetadata.TotalSupply = &totalSupply
	}

	asChainMetadata := false
	for _, chainM := range metadata.ChainsMetadatas {
		if chainM.ChainId == chainId {
			chainMetadata.ContractAddress = chainM.ContractAddress
			chainMetadata.Symbol = chainM.Symbol
			chainMetadata.Decimals = chainM.Decimals
			chainMetadata.IsOriginated = chainM.IsOriginated
			chainMetadata.TotalSupply = chainM.TotalSupply
			asChainMetadata = true
		}
	}
	if !asChainMetadata {
		metadata.ChainsMetadatas = append(metadata.ChainsMetadatas, chainMetadata)
	}
	k.bankKeeper.SetDenomMetaData(ctx, metadata)
	return token
}

func (k *Keeper) MintToken(ctx sdk.Context, hyperionId uint64, tokenAddress common.Address, amount math.Int, receiver common.Address) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	token, found := k.GetTokenFromAddress(ctx, hyperionId, tokenAddress)
	if !found {
		return errors.Wrap(types.ErrEmpty, "token not found")
	}

	if token.IsCosmosOriginated {
		return errors.Wrap(types.ErrEmpty, "token is cosmos originated")
	}

	k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(token.Denom, amount)))
	k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cmn.AccAddressFromHexAddress(receiver), sdk.NewCoins(sdk.NewCoin(token.Denom, amount)))

	return nil
}

func (k *Keeper) BurnToken(ctx sdk.Context, hyperionId uint64, tokenAddress common.Address, amount math.Int, sender common.Address) error {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, k.svcTags)
	defer doneFn()

	token, found := k.GetTokenFromAddress(ctx, hyperionId, tokenAddress)
	if !found {
		return errors.Wrap(types.ErrEmpty, "token not found")
	}

	if token.IsCosmosOriginated {
		return errors.Wrap(types.ErrEmpty, "token is cosmos originated")
	}

	k.bankKeeper.SendCoinsFromAccountToModule(ctx, cmn.AccAddressFromHexAddress(sender), types.ModuleName, sdk.NewCoins(sdk.NewCoin(token.Denom, amount)))
	k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(token.Denom, amount)))

	return nil
}

func (k *Keeper) sanitizeSymbol(symbol string) string {
	// Compile a regex for everything that is NOT in the pattern
	// Remove the anchor ^ and $ to keep only the content of the pattern
	allowedChars := "a-zA-Z0-9/:._-"

	// regex that matches all characters that are not in allowedChars
	reNotAllowed := regexp.MustCompile(fmt.Sprintf(`[^%s]`, allowedChars))

	// Replace the forbidden characters by empty
	finalSymbol := reNotAllowed.ReplaceAllString(symbol, "")

	if finalSymbol == "" {
		return symbol
	}
	if len(finalSymbol) > 127 {
		return finalSymbol[:127]
	}
	if len(finalSymbol) < 3 {
		return "HEL-" + symbol
	}
	return finalSymbol
}

func (k *Keeper) CreateOrLinkTokenToChain(ctx sdk.Context, chainId uint64, chainName string, token *types.TokenAddressToDenomWithGenesisInfos) *types.TokenAddressToDenom {
	tokenPair, ok := k.erc20Keeper.GetTokenPair(ctx, k.erc20Keeper.GetTokenPairID(ctx, token.TokenAddressToDenom.Denom))

	if !ok {

		coinMetadata := banktypes.Metadata{
			Description: fmt.Sprintf("Token %s created with Hyperion", token.TokenAddressToDenom.Denom),
			Base:        token.TokenAddressToDenom.Denom,
			Name:        token.TokenAddressToDenom.Symbol,
			Symbol:      token.TokenAddressToDenom.Symbol,
			Decimals:    uint32(token.TokenAddressToDenom.Decimals),
			Display:     token.TokenAddressToDenom.Symbol,
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    token.TokenAddressToDenom.Denom,
					Exponent: 0,
				},
				{
					Denom:    token.TokenAddressToDenom.Symbol,
					Exponent: uint32(token.TokenAddressToDenom.Decimals),
				},
			},
			Logo: token.Logo,
		}

		if err := coinMetadata.Validate(); err != nil {
			if strings.Contains(err.Error(), "invalid metadata display denom") {
				sanitizedSymbol := k.sanitizeSymbol(token.TokenAddressToDenom.Symbol)
				coinMetadata.Name = sanitizedSymbol
				coinMetadata.Symbol = sanitizedSymbol
				coinMetadata.Display = sanitizedSymbol
				coinMetadata.DenomUnits[1].Denom = sanitizedSymbol
				if err := coinMetadata.Validate(); err != nil {
					fmt.Println("error validating symbol after tried to sanitize it", err)
					return nil
				}
				fmt.Println("symbol sanitized", sanitizedSymbol)
			} else {
				fmt.Println("error", err)
				return nil
			}

		}

		contractAddr, err := k.erc20Keeper.DeployERC20Contract(ctx, coinMetadata)
		if err != nil {
			panic(fmt.Errorf("failed to deploy ERC20 contract: %w", err))
		}
		tokenPair = erc20types.NewTokenPair(contractAddr, token.TokenAddressToDenom.Denom, erc20types.OWNER_MODULE)
		k.erc20Keeper.SetToken(ctx, tokenPair)
		k.erc20Keeper.EnableDynamicPrecompiles(ctx, tokenPair.GetERC20Contract())

		// init one token for the module
		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.Coins{sdk.NewCoin(token.TokenAddressToDenom.Denom, math.NewInt(1))})
	}

	if token.TokenAddressToDenom.IsConcensusToken && !k.erc20Keeper.IsAssetWhitelisted(ctx, token.TokenAddressToDenom.Denom) {
		asset := erc20types.Asset{
			Denom:           token.TokenAddressToDenom.Denom,
			ContractAddress: tokenPair.Erc20Address,
			ChainId:         strconv.FormatUint(chainId, 10), // Exemple de chainId, à ajuster si nécessaire
			ChainName:       chainName,
			Decimals:        uint64(token.TokenAddressToDenom.Decimals),
			BaseWeight:      100, // Valeur par défaut, ajustable selon les besoins
			Symbol:          token.TokenAddressToDenom.Symbol,
		}
		k.erc20Keeper.AddAssetToConsensusWhitelist(ctx, asset)
	}

	for _, holder := range token.DefaultHolders {
		holder.Address = common.HexToAddress(holder.Address).Hex()

		k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.Coins{sdk.NewCoin(token.TokenAddressToDenom.Denom, holder.Amount)})
		k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cmn.AccAddressFromHexAddressString(holder.Address), sdk.Coins{sdk.NewCoin(token.TokenAddressToDenom.Denom, holder.Amount)})
	}
	return k.SetTokenToChainMetadata(ctx, chainId, token.TokenAddressToDenom)
}

func (k *Keeper) UpdateChainTokenLogo(ctx sdk.Context, chainId uint64, tokenAddress common.Address, logo string) error {
	denom, err := k.erc20Keeper.GetTokenDenom(ctx, tokenAddress)
	if err != nil {
		return err
	}
	metadata, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		return errors.Errorf("token metadata not found")
	}

	metadata.Logo = logo

	if err := metadata.Validate(); err != nil {
		return err
	}

	k.bankKeeper.SetDenomMetaData(ctx, metadata)
	return nil
}

func (k *Keeper) ValidateTokenMetaData(ctx sdk.Context, metadata *types.TokenMetadata) (*types.TokenMetadata, error) {

	if metadata == nil {
		return nil, errors.Errorf("undefined metadata")
	}

	if metadata.Decimals > 18 {
		return nil, errors.Errorf("claim data is not a valid Decimals: %v", metadata.Decimals)
	}

	if len(metadata.Name) > 100 {
		return nil, errors.Errorf("claim data is not a valid Name: %v len superior to 30", metadata.Name)
	}

	if len(metadata.Symbol) > 30 {
		return nil, errors.Errorf("claim data is not a valid Symbol: %v len superior to 30", metadata.Name)
	}

	return metadata, nil
}

func (k *Keeper) extractTokenMetadataInClaimDataWithDefault(ctx sdk.Context, claimData string, defaultMetadata *types.TokenMetadata) *types.TokenMetadata {
	if claimData == "" {
		return defaultMetadata
	}
	tokenMetadata, _, _ := k.parseClaimData(ctx, claimData)
	if tokenMetadata == nil {
		return defaultMetadata
	}

	if tokenMetadata.Symbol == "" {
		return defaultMetadata
	}
	return tokenMetadata
}

func (k *Keeper) parseClaimData(ctx sdk.Context, claimData string) (*types.TokenMetadata, *sdk.Msg, error) {
	var data types.ClaimData
	var msg sdk.Msg

	claimDataFull := claimData

	if err := json.Unmarshal([]byte(claimData), &data); err != nil {
		return nil, nil, nil
	}

	if data.Metadata != nil {
		claimDataFull = data.Data
		if _, err := k.ValidateTokenMetaData(ctx, data.Metadata); err != nil {
			return nil, nil, err
		}
		if claimDataFull == "" { // metadata alone
			return data.Metadata, nil, nil
		}
	}
	// Check if the claim data is a valid sdk msg
	// if err := k.cdc.UnmarshalInterfaceJSON([]byte(claimDataFull), &msg); err != nil {
	// 	return data.Metadata, nil, err
	// }

	return data.Metadata, &msg, nil
}

func (k *Keeper) handleValidateMsg(_ sdk.Context, msg *sdk.Msg) (bool, error) {
	switch (*msg).(type) {
	case *types.MsgSendToChain:
		return true, nil
	}
	return false, errors.Errorf("Message %s not managed", reflect.TypeOf(msg))
}

func (k *Keeper) GetHyperionContractBalance(ctx sdk.Context, hyperionId uint64, tokenContract common.Address) math.Int {
	denom, found := k.bankKeeper.GetDenomFromChainIdAndContractAddress(ctx, k.GetChainIdFromHyperionId(ctx, hyperionId), tokenContract.Hex())
	if !found {
		return math.ZeroInt()
	}
	metadata, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		return math.ZeroInt()
	}
	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.ChainId == k.GetChainIdFromHyperionId(ctx, hyperionId) && chainMetadata.TotalSupply != nil {
			return *chainMetadata.TotalSupply
		}
	}
	return math.ZeroInt()
}

func (k *Keeper) SetHyperionContractBalance(ctx sdk.Context, hyperionId uint64, tokenContract common.Address, balance math.Int) {
	denom, found := k.bankKeeper.GetDenomFromChainIdAndContractAddress(ctx, k.GetChainIdFromHyperionId(ctx, hyperionId), tokenContract.Hex())
	if !found {
		return
	}
	metadata, exists := k.bankKeeper.GetDenomMetaData(ctx, denom)
	if !exists {
		return
	}
	for _, chainMetadata := range metadata.ChainsMetadatas {
		if chainMetadata.ChainId == k.GetChainIdFromHyperionId(ctx, hyperionId) {
			chainMetadata.TotalSupply = &balance
		}
	}
	k.bankKeeper.SetDenomMetaData(ctx, metadata)
}
