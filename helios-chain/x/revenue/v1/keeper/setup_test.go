package keeper_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/ginkgo/v2"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"helios-core/helios-chain/app"
	utiltx "helios-core/helios-chain/testutil/tx"
	"helios-core/helios-chain/utils"
	evm "helios-core/helios-chain/x/evm/types"
	feemarkettypes "helios-core/helios-chain/x/feemarket/types"
	types "helios-core/helios-chain/x/revenue/v1/types"

	"github.com/stretchr/testify/suite"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx sdk.Context

	app            *app.HeliosApp
	queryClient    types.QueryClient
	queryClientEvm evm.QueryClient
	address        common.Address
	signer         keyring.Signer
	ethSigner      ethtypes.Signer
	consAddress    sdk.ConsAddress
	validator      stakingtypes.Validator
	denom          string
}

var s *KeeperTestSuite

var (
	contract = utiltx.GenerateAddress()
	deployer = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
	withdraw = sdk.AccAddress(utiltx.GenerateAddress().Bytes())
)

func TestKeeperTestSuite(t *testing.T) {
	s = new(KeeperTestSuite)
	suite.Run(t, s)

	// Run Ginkgo integration tests
	RegisterFailHandler(Fail)
	RunSpecs(t, "Keeper Suite")
}

func (suite *KeeperTestSuite) SetupTest() {
	chainID := utils.TestnetChainID + "-1"

	// Create a map of AppOptions
	appOpts := make(map[string]interface{})
	appOpts["chain-id"] = chainID
	appOpts["feemarket-genesis"] = feemarkettypes.DefaultGenesisState()

	suite.app = app.Setup(false, appOpts)
	suite.SetupApp(chainID)
}
