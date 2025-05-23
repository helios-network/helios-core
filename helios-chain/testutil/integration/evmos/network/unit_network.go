package network

import (
	"helios-core/helios-chain/app"
	"helios-core/helios-chain/x/evm/statedb"
	inflationtypes "helios-core/helios-chain/x/inflation/v1/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

// UnitTestNetwork is the implementation of the Network interface for unit tests.
// It embeds the IntegrationNetwork struct to reuse its methods and
// makes the App public for easier testing.
type UnitTestNetwork struct {
	IntegrationNetwork
	App *app.HeliosApp
}

var _ Network = (*UnitTestNetwork)(nil)

// NewUnitTestNetwork configures and initializes a new Evmos Network instance with
// the given configuration options. If no configuration options are provided
// it uses the default configuration.
//
// It panics if an error occurs.
// Note: Only uses for Unit Tests
func NewUnitTestNetwork(opts ...ConfigOption) *UnitTestNetwork {
	network := New(opts...)
	return &UnitTestNetwork{
		IntegrationNetwork: *network,
		App:                network.app,
	}
}

// GetStateDB returns the state database for the current block.
func (n *UnitTestNetwork) GetStateDB() *statedb.StateDB {
	headerHash := n.GetContext().HeaderHash()
	return statedb.New(
		n.GetContext(),
		n.App.EvmKeeper,
		statedb.NewEmptyTxConfig(common.BytesToHash(headerHash)),
	)
}

// FundAccount funds the given account with the given amount of coins.
func (n *UnitTestNetwork) FundAccount(addr sdktypes.AccAddress, coins sdktypes.Coins) error {
	ctx := n.GetContext()

	if err := n.app.BankKeeper.MintCoins(ctx, inflationtypes.ModuleName, coins); err != nil {
		return err
	}

	return n.app.BankKeeper.SendCoinsFromModuleToAccount(ctx, inflationtypes.ModuleName, addr, coins)
}
