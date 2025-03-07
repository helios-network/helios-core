package types_test

import (
	fmt "fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	hyperiontypes "helios-core/helios-chain/modules/hyperion/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	utiltx "helios-core/helios-chain/testutil/tx"
)

type MsgsTestSuite struct {
	suite.Suite
}

func TestMsgsTestSuite(t *testing.T) {
	suite.Run(t, new(MsgsTestSuite))
}
func (suite *MsgsTestSuite) TestMsgSetOrchestratorAddressesGetters() {
	msg := hyperiontypes.NewMsgSetOrchestratorAddress(
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
		sdk.AccAddress(utiltx.GenerateAddress().Bytes()),
		utiltx.GenerateAddress(),
	)
	err := msg.ValidateBasic()
	fmt.Println(err)
}
