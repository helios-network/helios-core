package keeper_test

import (
	"math/big"
	"time"

	"cosmossdk.io/math"
	//nolint:revive // dot imports are fine for Ginkgo
	. "github.com/onsi/gomega"

	"helios-core/helios-chain/crypto/ethsecp256k1"
	"helios-core/helios-chain/testutil"
	utiltx "helios-core/helios-chain/testutil/tx"
	"helios-core/helios-chain/utils"
	evmtypes "helios-core/helios-chain/x/evm/types"
	revtypes "helios-core/helios-chain/x/revenue/v1/types"

	"github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func (suite *KeeperTestSuite) SetupApp(chainID string) {
	t := suite.T()
	// account key
	priv, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.address = common.BytesToAddress(priv.PubKey().Address().Bytes())
	suite.signer = utiltx.NewSigner(priv)

	suite.denom = utils.BaseDenom

	// consensus key
	privCons, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	suite.consAddress = sdk.ConsAddress(privCons.PubKey().Address())

	suite.ctx = suite.app.BaseApp.NewContext(false)
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	revtypes.RegisterQueryServer(queryHelper, suite.app.RevenueKeeper)
	suite.queryClient = revtypes.NewQueryClient(queryHelper)

	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelperEvm)

	params := revtypes.DefaultParams()
	params.EnableRevenue = true
	err = suite.app.RevenueKeeper.SetParams(suite.ctx, params)
	require.NoError(t, err)

	stakingParams, err := suite.app.StakingKeeper.GetParams(suite.ctx)
	require.NoError(t, err)
	stakingParams.BondDenom = suite.denom
	err = suite.app.StakingKeeper.SetParams(suite.ctx, stakingParams)
	require.NoError(t, err)

	evmParams := suite.app.EvmKeeper.GetParams(suite.ctx)
	evmParams.EvmDenom = suite.denom
	err = suite.app.EvmKeeper.SetParams(suite.ctx, evmParams)
	require.NoError(t, err)

	inflationParams := suite.app.InflationKeeper.GetParams(suite.ctx)
	inflationParams.EnableInflation = false
	err = suite.app.InflationKeeper.SetParams(suite.ctx, inflationParams)
	require.NoError(t, err)

	// Set Validator
	valAddr := sdk.ValAddress(suite.address.Bytes())
	validator, err := stakingtypes.NewValidator(valAddr.String(), privCons.PubKey(), stakingtypes.Description{})
	require.NoError(t, err)

	// Update validator directly without using TestingUpdateValidator
	validator.Status = stakingtypes.Bonded
	validator.Tokens = math.NewInt(1000000)
	err = suite.app.StakingKeeper.SetValidator(suite.ctx, validator)
	require.NoError(t, err)
	err = suite.app.StakingKeeper.SetValidatorByConsAddr(suite.ctx, validator)
	require.NoError(t, err)
	err = suite.app.StakingKeeper.Hooks().AfterValidatorCreated(suite.ctx, valAddr)
	require.NoError(t, err)

	validators, err := suite.app.StakingKeeper.GetBondedValidatorsByPower(suite.ctx)
	require.NoError(t, err)
	suite.validator = validators[0]

	suite.ethSigner = ethtypes.LatestSignerForChainID(big.NewInt(1)) // Use chainID 1 for testing
}

// Commit commits and starts a new block with an updated context.
func (suite *KeeperTestSuite) Commit() {
	suite.CommitAfter(time.Second * 0)
}

// Commit commits a block at a given time.
func (suite *KeeperTestSuite) CommitAfter(t time.Duration) {
	var err error
	suite.ctx, err = testutil.CommitAndCreateNewCtx(suite.ctx, suite.app, t, nil)
	suite.Require().NoError(err)
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())

	revtypes.RegisterQueryServer(queryHelper, suite.app.RevenueKeeper)
	suite.queryClient = revtypes.NewQueryClient(queryHelper)

	queryHelperEvm := baseapp.NewQueryServerTestHelper(suite.ctx, suite.app.InterfaceRegistry())
	evmtypes.RegisterQueryServer(queryHelperEvm, suite.app.EvmKeeper)
	suite.queryClientEvm = evmtypes.NewQueryClient(queryHelperEvm)
}

func calculateFees(
	denom string,
	params revtypes.Params,
	res *types.ExecTxResult,
	gasPrice *big.Int,
) (sdk.Coin, sdk.Coin) {
	feeDistribution := math.NewInt(int64(res.GasUsed)).Mul(math.NewIntFromBigInt(gasPrice))
	developerFee := math.LegacyNewDecFromInt(feeDistribution).Mul(params.DeveloperShares)
	developerCoins := sdk.NewCoin(denom, developerFee.TruncateInt())
	validatorShares := math.LegacyOneDec().Sub(params.DeveloperShares)
	validatorFee := math.LegacyNewDecFromInt(feeDistribution).Mul(validatorShares)
	validatorCoins := sdk.NewCoin(denom, validatorFee.TruncateInt())
	return developerCoins, validatorCoins
}

// Global variable access functions
func getTestKeeper() *KeeperTestSuite {
	return s
}

func getNonce(addressBytes []byte) uint64 {
	keeper := getTestKeeper()
	return keeper.app.EvmKeeper.GetNonce(
		keeper.ctx,
		common.BytesToAddress(addressBytes),
	)
}

func registerFee(
	priv *ethsecp256k1.PrivKey,
	contractAddress *common.Address,
	withdrawerAddress sdk.AccAddress,
	nonces []uint64,
) *types.ExecTxResult {
	keeper := getTestKeeper()
	deployerAddress := sdk.AccAddress(priv.PubKey().Address())
	msg := revtypes.NewMsgRegisterRevenue(*contractAddress, deployerAddress, withdrawerAddress, nonces)

	res, err := testutil.DeliverTx(keeper.ctx, keeper.app, priv, nil, msg)
	keeper.Require().NoError(err)
	keeper.Commit()

	if res.IsOK() {
		registerEvent := res.GetEvents()[8]
		Expect(registerEvent.Type).To(Equal(revtypes.EventTypeRegisterRevenue))
		Expect(registerEvent.Attributes[0].Key).To(Equal(sdk.AttributeKeySender))
		Expect(registerEvent.Attributes[1].Key).To(Equal(revtypes.AttributeKeyContract))
		Expect(registerEvent.Attributes[2].Key).To(Equal(revtypes.AttributeKeyWithdrawerAddress))
	}
	return &res
}

func contractInteract(
	priv *ethsecp256k1.PrivKey,
	contractAddr *common.Address,
	gasPrice *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	data []byte,
	accesses *ethtypes.AccessList,
) *types.ExecTxResult {
	keeper := getTestKeeper()
	msgEthereumTx := buildEthTx(priv, contractAddr, gasPrice, gasFeeCap, gasTipCap, data, accesses)
	res, err := testutil.DeliverEthTx(keeper.app, priv, msgEthereumTx)
	Expect(err).To(BeNil())
	Expect(res.IsOK()).To(Equal(true), res.GetLog())
	return &res
}

func buildEthTx(
	priv *ethsecp256k1.PrivKey,
	to *common.Address,
	gasPrice *big.Int,
	gasFeeCap *big.Int,
	gasTipCap *big.Int,
	data []byte,
	accesses *ethtypes.AccessList,
) *evmtypes.MsgEthereumTx {
	chainID := big.NewInt(1) // Use chainID 1 for testing
	from := common.BytesToAddress(priv.PubKey().Address().Bytes())
	nonce := getNonce(from.Bytes())
	gasLimit := uint64(10000000)
	ethTxParams := evmtypes.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     nonce,
		To:        to,
		GasPrice:  gasPrice,
		GasLimit:  gasLimit,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Input:     data,
		Accesses:  accesses,
	}
	msgEthereumTx := evmtypes.NewTx(&ethTxParams)
	msgEthereumTx.From = from.String()
	return msgEthereumTx
}
