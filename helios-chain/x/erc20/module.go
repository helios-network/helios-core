package erc20

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/spf13/cobra"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	"helios-core/helios-chain/x/erc20/client/cli"
	"helios-core/helios-chain/x/erc20/keeper"
	"helios-core/helios-chain/x/erc20/types"
)

// consensusVersion defines the current x/erc20 module consensus version.
const consensusVersion = 4

// type check to ensure the interface is properly implemented
var (
	_ module.AppModule           = AppModule{}
	_ module.AppModuleBasic      = AppModuleBasic{}
	_ module.AppModuleSimulation = AppModule{}

	_ appmodule.AppModule   = AppModule{}
	_ module.HasABCIGenesis = AppModule{}

	_ appmodule.HasBeginBlocker = AppModule{}
	_ appmodule.HasEndBlocker   = AppModule{}
)

// app module Basics object
type AppModuleBasic struct{}

func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec performs a no-op as the erc20 doesn't support Amino encoding
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)
}

// ConsensusVersion returns the consensus state-breaking version for the module.
func (AppModuleBasic) ConsensusVersion() uint64 {
	return consensusVersion
}

// RegisterInterfaces registers interfaces and implementations of the erc20 module.
func (AppModuleBasic) RegisterInterfaces(interfaceRegistry codectypes.InterfaceRegistry) {
	types.RegisterInterfaces(interfaceRegistry)
}

// DefaultGenesis returns default genesis state as raw bytes for the erc20
// module.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesisState())
}

func (b AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, _ client.TxEncodingConfig, bz json.RawMessage) error {
	var genesisState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesisState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}

	return genesisState.Validate()
}

// RegisterRESTRoutes performs a no-op as the erc20 module doesn't expose REST
// endpoints
func (AppModuleBasic) RegisterRESTRoutes(_ client.Context, _ *mux.Router) {}

func (b AppModuleBasic) RegisterGRPCGatewayRoutes(c client.Context, serveMux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), serveMux, types.NewQueryClient(c)); err != nil {
		panic(err)
	}
}

// GetTxCmd returns the root tx command for the erc20 module.
func (b AppModuleBasic) GetTxCmd() *cobra.Command {
	return cli.NewTxCmd()
}

// GetTxCmd returns the root query command for the erc20 module.
func (b AppModuleBasic) GetQueryCmd() *cobra.Command {
	return cli.GetQueryCmd()
}

type AppModule struct {
	AppModuleBasic
	keeper keeper.Keeper
	ak     authkeeper.AccountKeeper
	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace types.Subspace
	bankKeeper     bankkeeper.Keeper
}

// NewAppModule creates a new AppModule Object
func NewAppModule(
	k keeper.Keeper,
	ak authkeeper.AccountKeeper,
	ss types.Subspace,
	bankKeeper bankkeeper.Keeper,
) AppModule {
	return AppModule{
		AppModuleBasic: AppModuleBasic{},
		keeper:         k,
		ak:             ak,
		legacySubspace: ss,
		bankKeeper:     bankKeeper,
	}
}

func (AppModule) Name() string {
	return types.ModuleName
}

func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), &am.keeper)
	types.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	// migrator := keeper.NewMigrator(am.keeper, am.legacySubspace)

	// NOTE: the migrations below will only run if the consensus version has changed
	// since the last release
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesisState types.GenesisState

	cdc.MustUnmarshalJSON(data, &genesisState)
	InitGenesis(ctx, am.keeper, am.ak, genesisState, am.bankKeeper)
	return []abci.ValidatorUpdate{}
}

func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(gs)
}

func (am AppModule) GenerateGenesisState(_ *module.SimulationState) {
}

func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {
}

func (am AppModule) WeightedOperations(_ module.SimulationState) []simtypes.WeightedOperation {
	return []simtypes.WeightedOperation{}
}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

func (am AppModule) BeginBlock(ctx context.Context) error {
	return nil
}

func (am AppModule) EndBlock(ctx context.Context) error {
	oldDynamicPrecompiles := am.keeper.GetOldDynamicPrecompiles(sdk.UnwrapSDKContext(ctx))
	oldNativePrecompiles := am.keeper.GetOldNativePrecompiles(sdk.UnwrapSDKContext(ctx))

	fmt.Println("oldDynamicPrecompiles", oldDynamicPrecompiles)
	fmt.Println("oldNativePrecompiles", oldNativePrecompiles)

	if len(oldDynamicPrecompiles) > 0 {
		for _, dynamicPrecompile := range oldDynamicPrecompiles {
			am.keeper.SetDynamicPrecompileEnabled(sdk.UnwrapSDKContext(ctx), common.HexToAddress(dynamicPrecompile))
		}
		am.keeper.SetOldDynamicPrecompiles(sdk.UnwrapSDKContext(ctx), []string{})
	}
	if len(oldNativePrecompiles) > 0 {
		for _, nativePrecompile := range oldNativePrecompiles {
			am.keeper.SetNativePrecompileEnabled(sdk.UnwrapSDKContext(ctx), common.HexToAddress(nativePrecompile))
		}
		am.keeper.SetOldNativePrecompiles(sdk.UnwrapSDKContext(ctx), []string{})
	}

	return nil
}
