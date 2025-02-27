// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/evmos/evmos/blob/main/LICENSE)
package backend

import (
	"fmt"
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	rpctypes "helios-core/helios-chain/rpc/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

	erc20types "helios-core/helios-chain/x/erc20/types"
)

func (b *Backend) GetHeliosAddress(address common.Address) (string, error) {
	return sdk.AccAddress(address.Bytes()).String(), nil
}

func (b *Backend) GetHeliosValoperAddress(address common.Address) (string, error) {
	return sdk.ValAddress(sdk.AccAddress(address.Bytes())).String(), nil
}

// GetCode returns the contract code at the given address and block number.
func (b *Backend) GetCode(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}

	req := &evmtypes.QueryCodeRequest{
		Address: address.String(),
	}

	res, err := b.queryClient.Code(rpctypes.ContextWithHeight(blockNum.Int64()), req)
	if err != nil {
		return nil, err
	}

	return res.Code, nil
}

// GetProof returns an account object with proof and any storage proofs
func (b *Backend) GetProof(address common.Address, storageKeys []string, blockNrOrHash rpctypes.BlockNumberOrHash) (*rpctypes.AccountResult, error) {
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}

	height := blockNum.Int64()

	_, err = b.TendermintBlockByNumber(blockNum)
	if err != nil {
		// the error message imitates geth behavior
		return nil, errors.New("header not found")
	}
	ctx := rpctypes.ContextWithHeight(height)

	// if the height is equal to zero, meaning the query condition of the block is either "pending" or "latest"
	if height == 0 {
		bn, err := b.BlockNumber()
		if err != nil {
			return nil, err
		}

		if bn > math.MaxInt64 {
			return nil, fmt.Errorf("not able to query block number greater than MaxInt64")
		}

		height = int64(bn) //#nosec G701 G115 -- checked for int overflow already
	}

	clientCtx := b.clientCtx.WithHeight(height)

	// query storage proofs
	storageProofs := make([]rpctypes.StorageResult, len(storageKeys))

	for i, key := range storageKeys {
		hexKey := common.HexToHash(key)
		valueBz, proof, err := b.queryClient.GetProof(clientCtx, evmtypes.StoreKey, evmtypes.StateKey(address, hexKey.Bytes()))
		if err != nil {
			return nil, err
		}

		storageProofs[i] = rpctypes.StorageResult{
			Key:   key,
			Value: (*hexutil.Big)(new(big.Int).SetBytes(valueBz)),
			Proof: GetHexProofs(proof),
		}
	}

	// query EVM account
	req := &evmtypes.QueryAccountRequest{
		Address: address.String(),
	}

	res, err := b.queryClient.Account(ctx, req)
	if err != nil {
		return nil, err
	}

	// query account proofs
	accountKey := bytes.HexBytes(append(authtypes.AddressStoreKeyPrefix, address.Bytes()...))
	_, proof, err := b.queryClient.GetProof(clientCtx, authtypes.StoreKey, accountKey)
	if err != nil {
		return nil, err
	}

	balance, ok := sdkmath.NewIntFromString(res.Balance)
	if !ok {
		return nil, errors.New("invalid balance")
	}

	return &rpctypes.AccountResult{
		Address:      address,
		AccountProof: GetHexProofs(proof),
		Balance:      (*hexutil.Big)(balance.BigInt()),
		CodeHash:     common.HexToHash(res.CodeHash),
		Nonce:        hexutil.Uint64(res.Nonce),
		StorageHash:  common.Hash{}, // NOTE: Evmos doesn't have a storage hash. TODO: implement?
		StorageProof: storageProofs,
	}, nil
}

// GetStorageAt returns the contract storage at the given address, block number, and key.
func (b *Backend) GetStorageAt(address common.Address, key string, blockNrOrHash rpctypes.BlockNumberOrHash) (hexutil.Bytes, error) {
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}

	req := &evmtypes.QueryStorageRequest{
		Address: address.String(),
		Key:     key,
	}

	res, err := b.queryClient.Storage(rpctypes.ContextWithHeight(blockNum.Int64()), req)
	if err != nil {
		return nil, err
	}

	value := common.HexToHash(res.Value)
	return value.Bytes(), nil
}

// GetBalance returns the provided account's balance up to the provided block number.
func (b *Backend) GetBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error) {
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}

	req := &evmtypes.QueryBalanceRequest{
		Address: address.String(),
	}

	_, err = b.TendermintBlockByNumber(blockNum)
	if err != nil {
		return nil, err
	}

	res, err := b.queryClient.Balance(rpctypes.ContextWithHeight(blockNum.Int64()), req)
	if err != nil {
		return nil, err
	}

	val, ok := sdkmath.NewIntFromString(res.Balance)
	if !ok {
		return nil, errors.New("invalid balance")
	}

	// balance can only be negative in case of pruned node
	if val.IsNegative() {
		return nil, errors.New("couldn't fetch balance. Node state is pruned")
	}

	return (*hexutil.Big)(val.BigInt()), nil
}

// GetTokenBalance returns specifical token balance for an account
func (b *Backend) GetTokenBalance(address common.Address, tokenAddress common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) (*hexutil.Big, error) {
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}

	balanceReq := &erc20types.QueryERC20BalanceOfRequest{
		Token:   tokenAddress.String(),
		Address: address.String(),
	}

	balanceRes, err := b.queryClient.Erc20.ERC20BalanceOf(rpctypes.ContextWithHeight(blockNum.Int64()), balanceReq)
	if err != nil {
		b.logger.Debug("failed to get ERC20 balance",
			"token", tokenAddress.String(),
			"error", err.Error())
		return nil, nil
	}

	val, ok := sdkmath.NewIntFromString(balanceRes.Balance)
	if !ok {
		b.logger.Debug("failed to parse ERC20 balance", "token", tokenAddress.String())
		return nil, nil
	}

	return (*hexutil.Big)(val.BigInt()), nil
}

// GetTokensBalance returns all token balances for an account
func (b *Backend) GetTokensBalance(address common.Address, blockNrOrHash rpctypes.BlockNumberOrHash) ([]rpctypes.TokenBalance, error) {
	balances := make([]rpctypes.TokenBalance, 0)

	// 2. Get Cosmos balances using bank query
	blockNum, err := b.BlockNumberFromTendermint(blockNrOrHash)
	if err != nil {
		return nil, err
	}
	// Get ERC20 balances using erc20 query
	// Create the query request
	erc20Req := &erc20types.QueryTokenPairsRequest{}
	erc20Res, err := b.queryClient.Erc20.TokenPairs(b.ctx, erc20Req)
	if err != nil {
		return nil, err
	}

	for _, token := range erc20Res.TokenPairs {
		// Query balance for each token
		balanceReq := &erc20types.QueryERC20BalanceOfRequest{
			Token:   token.Erc20Address,
			Address: address.String(),
		}

		balanceRes, err := b.queryClient.Erc20.ERC20BalanceOf(rpctypes.ContextWithHeight(blockNum.Int64()), balanceReq)
		if err != nil {
			b.logger.Debug("failed to get ERC20 balance",
				"token", token.Erc20Address,
				"error", err.Error())
			continue
		}

		val, ok := sdkmath.NewIntFromString(balanceRes.Balance)
		if !ok {
			b.logger.Debug("failed to parse ERC20 balance", "token", token.Denom)
			continue
		}

		balances = append(balances, rpctypes.TokenBalance{
			Address: common.HexToAddress(token.Erc20Address),
			Symbol:  token.Denom,
			Balance: (*hexutil.Big)(val.BigInt()),
		})
	}

	return balances, nil
}

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetTransactionCount(address common.Address, blockNum rpctypes.BlockNumber) (*hexutil.Uint64, error) {
	n := hexutil.Uint64(0)
	bn, err := b.BlockNumber()
	if err != nil {
		return &n, err
	}
	height := blockNum.Int64()

	currentHeight := int64(bn) //#nosec G701 G115 -- checked for int overflow already
	if height > currentHeight {
		return &n, errorsmod.Wrapf(
			sdkerrors.ErrInvalidHeight,
			"cannot query with height in the future (current: %d, queried: %d); please provide a valid height",
			currentHeight, height,
		)
	}
	// Get nonce (sequence) from account
	from := sdk.AccAddress(address.Bytes())
	accRet := b.clientCtx.AccountRetriever

	err = accRet.EnsureExists(b.clientCtx, from)
	if err != nil {
		// account doesn't exist yet, return 0
		return &n, nil
	}

	includePending := blockNum == rpctypes.EthPendingBlockNumber
	nonce, err := b.getAccountNonce(address, includePending, blockNum.Int64(), b.logger)
	if err != nil {
		return nil, err
	}

	n = hexutil.Uint64(nonce)
	return &n, nil
}

// GetTransactionCount returns the number of transactions at the given address up to the given block number.
func (b *Backend) GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	transactions := make([]*rpctypes.RPCTransaction, 0)

	// Get current block number using proper context
	latestBlockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, err
	}

	// Calculate starting position
	start := false
	counter := uint64(0)
	currentBlockNum := uint64(latestBlockNumber)

	// Get pending transactions first
	pendingTxs, err := b.PendingTransactions()
	if err == nil && len(pendingTxs) > 0 {
		for _, tx := range pendingTxs {
			msg, err := evmtypes.UnwrapEthereumMsgTx(tx)
			if err != nil {
				// not ethereum tx
				continue
			}
			rpctx, err := rpctypes.NewTransactionFromMsg(
				msg,
				common.Hash{},
				uint64(0),
				uint64(0),
				nil,
				b.chainID,
			)
			if err != nil {
				return nil, err
			}
			if rpctx.From == address || (rpctx.To != nil && *rpctx.To == address) {
				counter++
				if counter >= (uint64(page)-1)*uint64(size) {
					start = true
				}
				if start {
					transactions = append(transactions, rpctx)
					if uint64(len(transactions)) >= uint64(size) {
						return transactions, nil
					}
				}
			}
		}
	}

	// Iterate through blocks from latest to first
	for currentBlockNum > 0 && uint64(len(transactions)) < uint64(size) {
		block, err := b.GetBlockByNumber(rpctypes.BlockNumber(currentBlockNum), true)
		if err != nil {
			b.logger.Error("failed to get block", "number", currentBlockNum, "error", err.Error())
			break
		}
		// Check each transaction in the block
		if txs, ok := block["transactions"].([]interface{}); ok {
			for _, tx := range txs {
				if ethTx, ok := tx.(*rpctypes.RPCTransaction); ok {
					// Check if the address is either the sender or receiver
					if ethTx.From == address || (ethTx.To != nil && *ethTx.To == address) {
						counter++
						if counter >= (uint64(page)-1)*uint64(size) {
							start = true
						}
						if start {
							transactions = append(transactions, ethTx)
							if uint64(len(transactions)) >= uint64(size) {
								return transactions, nil
							}
						}
					}
				}
			}
		}
		currentBlockNum--
	}
	return transactions, nil
}
