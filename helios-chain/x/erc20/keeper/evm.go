package keeper

import (
	"math/big"

	evmtypes "helios-core/helios-chain/x/evm/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"helios-core/helios-chain/contracts"
	"helios-core/helios-chain/x/erc20/types"
)

// DeployERC20Contract creates and deploys an ERC20 contract on the EVM with the
// erc20 module account as owner.
func (k Keeper) DeployERC20Contract(
	ctx sdk.Context,
	coinMetadata banktypes.Metadata,
) (common.Address, error) {
	decimals := uint8(0)
	if len(coinMetadata.DenomUnits) > 0 {
		decimalsIdx := len(coinMetadata.DenomUnits) - 1
		decimals = uint8(coinMetadata.DenomUnits[decimalsIdx].Exponent) //#nosec G115
	}
	ctorArgs, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack(
		"",
		coinMetadata.Name,
		coinMetadata.Symbol,
		decimals,
	)
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(types.ErrABIPack, "coin metadata is invalid %s: %s", coinMetadata.Name, err.Error())
	}

	data := make([]byte, len(contracts.ERC20MinterBurnerDecimalsContract.Bin)+len(ctorArgs))
	copy(data[:len(contracts.ERC20MinterBurnerDecimalsContract.Bin)], contracts.ERC20MinterBurnerDecimalsContract.Bin)
	copy(data[len(contracts.ERC20MinterBurnerDecimalsContract.Bin):], ctorArgs)

	nonce, err := k.accountKeeper.GetSequence(ctx, types.ModuleAddress.Bytes())
	if err != nil {
		return common.Address{}, err
	}

	contractAddr := crypto.CreateAddress(types.ModuleAddress, nonce)
	_, err = k.evmKeeper.CallEVMWithData(ctx, types.ModuleAddress, nil, data, true)
	if err != nil {
		return common.Address{}, errorsmod.Wrapf(err, "failed to deploy contract for %s", coinMetadata.Name)
	}

	return contractAddr, nil
}

func (k Keeper) MintERC20Tokens(
	ctx sdk.Context,
	contractAddr common.Address,
	recipient common.Address,
	amount *big.Int,
) error {
	// Pack the arguments for the mint call: mint(address to, uint256 amount)
	mintData, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack("mint", recipient, amount)
	if err != nil {
		return errorsmod.Wrap(err, "failed to pack mint call data")
	}

	// Execute the call from the module account (which has the MINTER_ROLE) to the contract
	_, err = k.evmKeeper.CallEVMWithData(ctx, types.ModuleAddress, &contractAddr, mintData, true)
	if err != nil {
		return errorsmod.Wrap(err, "failed to mint tokens")
	}

	return nil
}

// DoesERC20ContractExist checks if an ERC20 contract exists on the EVM by validating a call to its `totalSupply` method.
func (k Keeper) DoesERC20ContractExist(
	ctx sdk.Context,
	contract common.Address,
) (bool, error) {
	// Use the ABI of the ERC20 contract
	erc20ABI := contracts.ERC20MinterBurnerDecimalsContract.ABI

	// Call the `totalSupply` function on the contract
	res, err := k.evmKeeper.CallEVM(ctx, erc20ABI, types.ModuleAddress, contract, false, "totalSupply")
	if err != nil {
		// If the call fails, it indicates the contract may not exist or is not an ERC20
		return false, nil
	}

	// Attempt to unpack the result to ensure the contract responds correctly
	unpacked, err := erc20ABI.Unpack("totalSupply", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return false, nil
	}

	// If unpacked successfully and the response is valid, the contract exists
	_, ok := unpacked[0].(*big.Int)
	return ok, nil
}

// QueryERC20 returns the data of a deployed ERC20 contract
func (k Keeper) QueryERC20(
	ctx sdk.Context,
	contract common.Address,
) (types.ERC20Data, error) {
	var (
		nameRes    types.ERC20StringResponse
		symbolRes  types.ERC20StringResponse
		decimalRes types.ERC20Uint8Response
	)

	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

	// Name
	res, err := k.evmKeeper.CallEVM(ctx, erc20, types.ModuleAddress, contract, false, "name")
	if err != nil {
		return types.ERC20Data{}, err
	}

	if err := erc20.UnpackIntoInterface(&nameRes, "name", res.Ret); err != nil {
		return types.ERC20Data{}, errorsmod.Wrapf(
			types.ErrABIUnpack, "failed to unpack name: %s", err.Error(),
		)
	}

	// Symbol
	res, err = k.evmKeeper.CallEVM(ctx, erc20, types.ModuleAddress, contract, false, "symbol")
	if err != nil {
		return types.ERC20Data{}, err
	}

	if err := erc20.UnpackIntoInterface(&symbolRes, "symbol", res.Ret); err != nil {
		return types.ERC20Data{}, errorsmod.Wrapf(
			types.ErrABIUnpack, "failed to unpack symbol: %s", err.Error(),
		)
	}

	// Decimals
	res, err = k.evmKeeper.CallEVM(ctx, erc20, types.ModuleAddress, contract, false, "decimals")
	if err != nil {
		return types.ERC20Data{}, err
	}

	if err := erc20.UnpackIntoInterface(&decimalRes, "decimals", res.Ret); err != nil {
		return types.ERC20Data{}, errorsmod.Wrapf(
			types.ErrABIUnpack, "failed to unpack decimals: %s", err.Error(),
		)
	}

	return types.NewERC20Data(nameRes.Value, symbolRes.Value, decimalRes.Value), nil
}

// BalanceOf queries an account's balance for a given ERC20 contract
func (k Keeper) BalanceOf(
	ctx sdk.Context,
	abi abi.ABI,
	contract, account common.Address,
) *big.Int {
	res, err := k.evmKeeper.CallEVM(ctx, abi, types.ModuleAddress, contract, false, "balanceOf", account)
	if err != nil {
		return nil
	}

	unpacked, err := abi.Unpack("balanceOf", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return nil
	}

	balance, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil
	}

	return balance
}

// monitorApprovalEvent returns an error if the given transactions logs include
// an unexpected `Approval` event
func (k Keeper) monitorApprovalEvent(res *evmtypes.MsgEthereumTxResponse) error {
	if res == nil || len(res.Logs) == 0 {
		return nil
	}

	logApprovalSig := []byte("Approval(address,address,uint256)")
	logApprovalSigHash := crypto.Keccak256Hash(logApprovalSig)

	for _, log := range res.Logs {
		if log.Topics[0] == logApprovalSigHash.Hex() {
			return errorsmod.Wrapf(
				types.ErrUnexpectedEvent, "unexpected Approval event",
			)
		}
	}

	return nil
}

// TotalSupply queries the total supply for a given ERC20 contract
func (k Keeper) TotalSupply(
	ctx sdk.Context,
	contract common.Address,
) (*big.Int, error) {
	erc20 := contracts.ERC20MinterBurnerDecimalsContract.ABI

	res, err := k.evmKeeper.CallEVM(ctx, erc20, types.ModuleAddress, contract, false, "totalSupply")
	if err != nil {
		return nil, err
	}

	unpacked, err := erc20.Unpack("totalSupply", res.Ret)
	if err != nil {
		return nil, errorsmod.Wrapf(
			types.ErrABIUnpack, "failed to unpack totalSupply: %s", err.Error(),
		)
	}

	if len(unpacked) == 0 {
		return nil, errorsmod.Wrap(types.ErrABIUnpack, "empty response")
	}

	totalSupply, ok := unpacked[0].(*big.Int)
	if !ok {
		return nil, errorsmod.Wrap(types.ErrABIUnpack, "invalid response type")
	}

	return totalSupply, nil
}
