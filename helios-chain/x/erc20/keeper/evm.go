package keeper

import (
	"fmt"
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
func (k *Keeper) DeployERC20Contract(
	ctx sdk.Context,
	coinMetadata banktypes.Metadata,
	authorities ...common.Address, // Variadic parameters (optional)
) (common.Address, error) {
	decimals := uint8(0)
	if len(coinMetadata.DenomUnits) > 0 {
		decimalsIdx := len(coinMetadata.DenomUnits) - 1
		decimals = uint8(coinMetadata.DenomUnits[decimalsIdx].Exponent) //#nosec G115
	}

	// Determine authorities based on provided parameters
	var initialOwner, mintAuthority, pauseAuthority, burnAuthority common.Address

	switch len(authorities) {
	case 0:
		// LEGACY BEHAVIOR: Module has all roles (as before)
		initialOwner = types.ModuleAddress
		mintAuthority = types.ModuleAddress
		pauseAuthority = types.ModuleAddress
		burnAuthority = types.ModuleAddress
	case 4:
		// NEW BEHAVIOR: Specific roles
		initialOwner = authorities[0]
		mintAuthority = authorities[1]
		pauseAuthority = authorities[2]
		burnAuthority = authorities[3]
	default:
		return common.Address{}, fmt.Errorf("invalid number of authorities: expected 0 or 4, got %d", len(authorities))
	}

	ctorArgs, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Pack(
		"",
		coinMetadata.Name,
		coinMetadata.Symbol,
		decimals,
		initialOwner,
		mintAuthority,
		pauseAuthority,
		burnAuthority,
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

	// Set the PairContract field in the denom metadata
	k.bankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
		Description:     coinMetadata.Description,
		DenomUnits:      coinMetadata.DenomUnits,
		Base:            coinMetadata.Base,
		Display:         coinMetadata.Display,
		Name:            coinMetadata.Name,
		Symbol:          coinMetadata.Symbol,
		URI:             coinMetadata.URI,
		URIHash:         coinMetadata.URIHash,
		Decimals:        coinMetadata.Decimals,
		Logo:            coinMetadata.Logo,
		ContractAddress: contractAddr.Hex(),
		ChainsMetadatas: coinMetadata.ChainsMetadatas,
	})

	return contractAddr, nil
}

func (k Keeper) MintERC20Tokens(
	ctx sdk.Context,
	contractAddr common.Address,
	recipient common.Address,
	amount *big.Int,
) error {
	// Use CallEVM instead of CallEVMWithData for better gas handling
	_, err := k.evmKeeper.CallEVM(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress, // from (module has MINTER_ROLE)
		contractAddr,        // contract address
		true,                // commit = true (state-changing call)
		"mint",              // method name
		recipient,           // to address
		amount,              // amount to mint
	)
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

// RevokeTempMinterRole revokes the temporary MINTER_ROLE from module after initial mint
// FIXED: Uses CallEVM instead of CallEVMWithData for proper gas estimation
func (k Keeper) RevokeTempMinterRole(
	ctx sdk.Context,
	contractAddr common.Address,
) error {
	minterRoleHash := crypto.Keccak256Hash([]byte("MINTER_ROLE"))

	// Use CallEVM instead of CallEVMWithData - it handles gas better
	_, err := k.evmKeeper.CallEVM(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress, // from (module has admin role)
		contractAddr,        // contract address
		true,                // commit = true (this is a state-changing call)
		"revokeRole",        // method name
		minterRoleHash,      // role hash (bytes32)
		types.ModuleAddress, // account to revoke from (module itself)
	)
	if err != nil {
		return errorsmod.Wrap(err, "failed to revoke temp MINTER_ROLE from module")
	}

	return nil
}

// SetMintAuthority grants MINTER_ROLE to a specific address
func (k Keeper) SetMintAuthority(
	ctx sdk.Context,
	contractAddr common.Address,
	mintAuthority common.Address,
) error {
	return k.grantRole(ctx, contractAddr, "MINTER_ROLE", mintAuthority)
}

// SetPauseAuthority grants PAUSER_ROLE to a specific address
func (k Keeper) SetPauseAuthority(
	ctx sdk.Context,
	contractAddr common.Address,
	pauseAuthority common.Address,
) error {
	return k.grantRole(ctx, contractAddr, "PAUSER_ROLE", pauseAuthority)
}

// SetBurnAuthority grants BURNER_ROLE to a specific address
func (k Keeper) SetBurnAuthority(
	ctx sdk.Context,
	contractAddr common.Address,
	burnAuthority common.Address,
) error {
	return k.grantRole(ctx, contractAddr, "BURNER_ROLE", burnAuthority)
}

// grantRole is a helper function to grant any role to an address
// FIXED: Uses CallEVM instead of CallEVMWithData
func (k Keeper) grantRole(
	ctx sdk.Context,
	contractAddr common.Address,
	role string,
	account common.Address,
) error {
	// Hash the role name
	roleHash := crypto.Keccak256Hash([]byte(role))

	// Use CallEVM for better gas estimation
	_, err := k.evmKeeper.CallEVM(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress, // from (module has DEFAULT_ADMIN_ROLE)
		contractAddr,        // contract address
		true,                // commit = true (state-changing call)
		"grantRole",         // method name
		roleHash,            // role hash (bytes32)
		account,             // account to grant role to
	)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to grant %s to %s", role, account.Hex())
	}

	return nil
}

// HasRole checks if an address has a specific role
func (k Keeper) HasRole(
	ctx sdk.Context,
	contractAddr common.Address,
	role string,
	account common.Address,
) (bool, error) {
	roleHash := crypto.Keccak256Hash([]byte(role))

	// hasRole(bytes32 role, address account) returns (bool)
	res, err := k.evmKeeper.CallEVM(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress,
		contractAddr,
		false,
		"hasRole",
		roleHash,
		account,
	)
	if err != nil {
		return false, err
	}

	unpacked, err := contracts.ERC20MinterBurnerDecimalsContract.ABI.Unpack("hasRole", res.Ret)
	if err != nil || len(unpacked) == 0 {
		return false, err
	}

	hasRole, ok := unpacked[0].(bool)
	if !ok {
		return false, errorsmod.Wrap(types.ErrABIUnpack, "failed to unpack hasRole result")
	}

	return hasRole, nil
}

// RevokeRole revokes a role from an address
// FIXED: Uses CallEVM instead of CallEVMWithData
func (k Keeper) RevokeRole(
	ctx sdk.Context,
	contractAddr common.Address,
	role string,
	account common.Address,
) error {
	roleHash := crypto.Keccak256Hash([]byte(role))

	// Use CallEVM for better gas estimation
	_, err := k.evmKeeper.CallEVM(
		ctx,
		contracts.ERC20MinterBurnerDecimalsContract.ABI,
		types.ModuleAddress, // from (module has admin role)
		contractAddr,        // contract address
		true,                // commit = true (state-changing call)
		"revokeRole",        // method name
		roleHash,            // role hash (bytes32)
		account,             // account to revoke role from
	)
	if err != nil {
		return errorsmod.Wrapf(err, "failed to revoke %s from %s", role, account.Hex())
	}

	return nil
}
