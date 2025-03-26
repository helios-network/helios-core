package feedistribution

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/ethereum/go-ethereum/common"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	evmtypes "helios-core/helios-chain/x/evm/types"
	"helios-core/helios-chain/x/feedistribution/client/cli"
	"helios-core/helios-chain/x/feedistribution/keeper"
	feedtypes "helios-core/helios-chain/x/feedistribution/types"
)

var (
	_ module.AppModule      = AppModule{}
	_ module.AppModuleBasic = AppModuleBasic{}
)

// AppModuleBasic defines the basic application module used by the feedistribution module.
type AppModuleBasic struct{}

// Name returns the feedistribution module's name.
func (AppModuleBasic) Name() string {
	return feedtypes.ModuleName
}

// RegisterLegacyAminoCodec registers the feedistribution module's types on the given LegacyAmino codec.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	feedtypes.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers the module's interface types
func (AppModuleBasic) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	feedtypes.RegisterInterfaces(registry)
}

// DefaultGenesis returns default genesis state as raw bytes for the feedistribution module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(feedtypes.DefaultGenesis())
}

// ValidateGenesis performs genesis state validation for the feedistribution module.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var data feedtypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &data); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", feedtypes.ModuleName, err)
	}

	return data.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the feedistribution module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := feedtypes.RegisterQueryHandlerClient(context.Background(), mux, feedtypes.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the feedistribution module.
func (AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.GetTxCmd()
}

// GetQueryCmd returns the root query command for the feedistribution module.
func (AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

// AppModule implements the AppModule interface for the feedistribution module.
type AppModule struct {
	AppModuleBasic

	keeper keeper.Keeper
	cdc    codec.Codec
}

// NewAppModule creates a new AppModule object
func NewAppModule(k keeper.Keeper, cdc codec.Codec) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		cdc:            cdc,
	}
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// RegisterInvariants registers the feedistribution module's invariants.
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// RegisterServices registers a GRPC query service to respond to the module-specific GRPC queries.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	feedtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	feedtypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) []abci.ValidatorUpdate {
	var genState feedtypes.GenesisState
	cdc.MustUnmarshalJSON(gs, &genState)

	// Initialize the module's state
	if err := genState.Validate(); err != nil {
		panic(fmt.Errorf("failed to validate %s genesis state: %w", feedtypes.ModuleName, err))
	}

	// Set module parameters
	if err := am.keeper.SetParams(ctx, genState.Params); err != nil {
		panic(fmt.Errorf("failed to set %s module parameters: %w", feedtypes.ModuleName, err))
	}

	// Initialize block fees
	for _, blockFee := range genState.BlockFees {
		contractAddr := common.HexToAddress(blockFee.ContractAddress)
		if err := am.keeper.SetBlockFees(ctx, contractAddr, blockFee); err != nil {
			panic(fmt.Errorf("failed to set block fees for contract %s: %w", blockFee.ContractAddress, err))
		}
	}

	// Initialize contracts
	for _, contract := range genState.Contracts {
		if err := am.keeper.SetContractInfo(ctx, contract); err != nil {
			panic(fmt.Errorf("failed to set contract info for %s: %w", contract.ContractAddress, err))
		}
	}

	return []abci.ValidatorUpdate{}
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	var genState feedtypes.GenesisState

	// Export parameters
	genState.Params = am.keeper.GetParams(ctx)

	// Export block fees
	var blockFees []feedtypes.BlockFees
	am.keeper.IterateBlockFees(ctx, func(contract common.Address, fees feedtypes.BlockFees) bool {
		blockFees = append(blockFees, fees)
		return false
	})
	genState.BlockFees = blockFees

	// Export contracts
	var contracts []feedtypes.ContractInfo
	am.keeper.IterateRevenues(ctx, func(contract common.Address, revenue feedtypes.Revenue) bool {
		contractInfo, found := am.keeper.GetContractInfo(ctx, contract.String())
		if found {
			contracts = append(contracts, contractInfo)
		}
		return false
	})
	genState.Contracts = contracts

	return cdc.MustMarshalJSON(&genState)
}

// ConsensusVersion implements AppModule/ConsensusVersion.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block
func (am AppModule) BeginBlock(ctx context.Context) error {
	// No-op for now
	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block
func (am AppModule) EndBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := am.keeper.Logger(sdkCtx)

	// Get module parameters
	params := am.keeper.GetParams(sdkCtx)
	if !params.EnableFeeDistribution {
		return nil
	}

	// Track total fees distributed in this block
	totalFeesDistributed := sdk.NewCoins()

	// Iterate over all block fees
	am.keeper.IterateBlockFees(sdkCtx, func(contract common.Address, blockFees feedtypes.BlockFees) bool {
		// Get contract info
		contractInfo, found := am.keeper.GetContractInfo(sdkCtx, contract.String())
		if !found {
			logger.Debug(
				"contract not found for fee distribution",
				"contract", contract.String(),
			)
			return false
		}

		// Get withdrawer address (defaults to deployer if not set)
		withdrawerAddr := contractInfo.DeployerAddress
		if contractInfo.WithdrawerAddress != "" {
			withdrawerAddr = contractInfo.WithdrawerAddress
		}

		withdrawer, err := sdk.AccAddressFromBech32(withdrawerAddr)
		if err != nil {
			logger.Error(
				"invalid withdrawer address",
				"contract", contract.String(),
				"withdrawer", withdrawerAddr,
				"error", err,
			)
			return false
		}

		// Calculate developer's share of fees
		developerFee := params.DeveloperShares.MulInt(blockFees.AccumulatedFees).TruncateInt()
		fees := sdk.NewCoins(sdk.NewCoin(evmtypes.GetEVMCoinDenom(), developerFee))

		// Distribute fees to the contract deployer/withdrawer
		err = am.keeper.DistributeFees(sdkCtx, withdrawer, fees)
		if err != nil {
			logger.Error(
				"failed to distribute fees",
				"contract", contract.String(),
				"withdrawer", withdrawerAddr,
				"amount", fees.String(),
				"error", err,
			)
			return false
		}

		totalFeesDistributed = totalFeesDistributed.Add(fees...)

		// Emit event for fee distribution
		sdkCtx.EventManager().EmitEvents(
			sdk.Events{
				sdk.NewEvent(
					feedtypes.EventTypeDistributeFees,
					sdk.NewAttribute(feedtypes.AttributeKeyContract, contract.String()),
					sdk.NewAttribute(feedtypes.AttributeKeyDeployer, contractInfo.DeployerAddress),
					sdk.NewAttribute(feedtypes.AttributeKeyWithdrawer, withdrawerAddr),
					sdk.NewAttribute(sdk.AttributeKeyAmount, fees.String()),
				),
			},
		)

		// Clear accumulated fees for this contract
		am.keeper.ClearBlockFees(sdkCtx, contract)

		return false
	})

	// Emit event for total fees distributed in this block
	if !totalFeesDistributed.IsZero() {
		sdkCtx.EventManager().EmitEvents(
			sdk.Events{
				sdk.NewEvent(
					feedtypes.EventTypeDistributeFees,
					sdk.NewAttribute("total_fees_distributed", totalFeesDistributed.String()),
					sdk.NewAttribute("block_height", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
				),
			},
		)
	}

	return nil
}
