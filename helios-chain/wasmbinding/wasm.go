package wasmbinding

import (
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	auctionkeeper "helios-core/helios-chain/modules/auction/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"

	wasmxkeeper "helios-core/helios-chain/modules/wasmx/keeper"

	exchangekeeper "helios-core/helios-chain/modules/exchange/keeper"
	oraclekeeper "helios-core/helios-chain/modules/oracle/keeper"
	tokenfactorykeeper "helios-core/helios-chain/modules/tokenfactory/keeper"
)

func RegisterCustomPlugins(
	authzKeeper *authzkeeper.Keeper,
	bankBaseKeeper bankkeeper.BaseKeeper,
	auctionKeeper *auctionkeeper.Keeper,
	exchangeKeeper *exchangekeeper.Keeper,
	feegrantKeeper *feegrantkeeper.Keeper,
	oracleKeeper *oraclekeeper.Keeper,
	tokenFactoryKeeper *tokenfactorykeeper.Keeper,
	wasmxKeeper *wasmxkeeper.Keeper,
	router wasmkeeper.MessageRouter,
) []wasmkeeper.Option {
	wasmQueryPlugin := NewQueryPlugin(authzKeeper, auctionKeeper, exchangeKeeper, oracleKeeper, &bankBaseKeeper, tokenFactoryKeeper, wasmxKeeper, feegrantKeeper)

	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Custom: CustomQuerier(wasmQueryPlugin),
	})

	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(router, bankBaseKeeper, exchangeKeeper, tokenFactoryKeeper),
	)

	return []wasmkeeper.Option{
		queryPluginOpt,
		messengerDecoratorOpt,
	}
}

func RegisterStargateQueries(queryRouter baseapp.GRPCQueryRouter, codecInterface codec.Codec) []wasmkeeper.Option {
	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: StargateQuerier(queryRouter, codecInterface),
	})

	return []wasmkeeper.Option{
		queryPluginOpt,
	}
}
