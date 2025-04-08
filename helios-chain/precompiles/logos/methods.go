package logos

import (
	"fmt"

	cmn "helios-core/helios-chain/precompiles/common"

	"helios-core/helios-chain/x/evm/core/vm"
	logoskeeper "helios-core/helios-chain/x/logos/keeper"

	logostypes "helios-core/helios-chain/x/logos/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const (
	UploadLogoMethod = "uploadLogo"
)

func (p Precompile) UploadLogo(
	ctx sdk.Context,
	origin common.Address,
	contract *vm.Contract,
	stateDB vm.StateDB,
	method *abi.Method,
	args []interface{},
) ([]byte, error) {

	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	logoBase64, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid string LogoBase64")
	}

	msg := &logostypes.MsgStoreLogoRequest{
		Creator: cmn.AccAddressFromHexAddress(origin).String(),
		Data:    logoBase64,
	}

	msgSrv := logoskeeper.NewMsgServerImpl(p.logosKeeper)
	resp, err := msgSrv.StoreLogo(ctx, msg)
	if err != nil {
		return nil, err
	}

	if err := p.EmitLogoUploadedEvent(ctx, stateDB, origin, p.Address(), resp.Hash); err != nil {
		return nil, err
	}

	fmt.Println("Logo uploaded", resp.Hash)

	return method.Outputs.Pack(true)
}
