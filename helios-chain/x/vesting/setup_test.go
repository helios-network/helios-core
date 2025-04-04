package vesting_test

import (
	"testing"

	"helios-core/helios-chain/app"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type VestingTestSuite struct {
	suite.Suite

	ctx        sdk.Context
	app        *app.HeliosApp
	address    common.Address
	validators []stakingtypes.Validator
	privKey    cryptotypes.PrivKey
	signer     keyring.Signer
}

var s *VestingTestSuite

func TestVestingTestSuite(t *testing.T) {
	s = new(VestingTestSuite)
	suite.Run(t, s)
}

func (s *VestingTestSuite) SetupTest() {
	s.DoSetupTest()
}
