package backend

import (
	"fmt"
	"math"
	"math/big"

	errorsmod "cosmossdk.io/errors"

	rpctypes "helios-core/helios-chain/rpc/types"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/bytes"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/pkg/errors"

	erc20types "helios-core/helios-chain/x/erc20/types"
)

func (b *Backend) GetHeliosAddress(address common.Address) (string, error) {
	return sdk.AccAddress(address.Bytes()).String(), nil
}

func (b *Backend) GetHeliosValoperAddress(address common.Address) (string, error) {
	return sdk.ValAddress(sdk.AccAddress(address.Bytes())).String(), nil
}

func (b *Backend) GetAccountType(address common.Address) (string, error) {
	if _, err := b.GetCronByAddress(address); err == nil {
		return "CRON", nil
	}
	for _, addr := range evmtypes.AvailableStaticPrecompiles {
		if addr == address.String() {
			return "PRECOMPILE", nil
		}
	}
	req := &evmtypes.QueryCodeRequest{
		Address: address.String(),
	}
	res, err := b.queryClient.Code(b.ctx, req)
	if err == nil && len(res.Code) != 0 {
		return "SMART_CONTRACT", nil
	}
	return "ADDRESS", nil
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
			Address:   common.HexToAddress(token.Erc20Address),
			Symbol:    token.Denom,
			Balance:   (*hexutil.Big)(val.BigInt()),
			BalanceUI: val.String(),
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

func convertChronosRPCTransactionToRPCTransaction(tx *chronostypes.CronTransactionRPC) *rpctypes.RPCTransaction {
	hash := common.HexToHash(tx.BlockHash)
	blockNumber := hexutil.Big(*hexutil.MustDecodeBig(tx.BlockNumber))
	gasPrice := hexutil.Big(*hexutil.MustDecodeBig(tx.GasPrice))
	gasFeeCap := hexutil.Big(*big.NewInt(0))
	gasTipCap := hexutil.Big(*big.NewInt(0))
	to := common.HexToAddress(tx.To)
	txIndex := hexutil.Uint64(hexutil.MustDecodeUint64(tx.TransactionIndex))
	chainId := hexutil.Big(*hexutil.MustDecodeBig(tx.ChainId))
	value := hexutil.Big(*hexutil.MustDecodeBig(tx.Value))
	v := hexutil.Big(*big.NewInt(0))
	r := hexutil.Big(*big.NewInt(0))
	s := hexutil.Big(*big.NewInt(0))

	return &rpctypes.RPCTransaction{
		BlockHash:        &hash,                                              // Convert string to common.Hash
		BlockNumber:      &blockNumber,                                       // Convert string to hexutil.Big
		From:             common.HexToAddress(tx.From),                       // Convert string to common.Address
		Gas:              hexutil.Uint64(hexutil.MustDecodeUint64(tx.Gas)),   // Convert string to hexutil.Uint64
		GasPrice:         &gasPrice,                                          // Convert string to hexutil.Big
		GasFeeCap:        &gasFeeCap,                                         // Convert string to hexutil.Big
		GasTipCap:        &gasTipCap,                                         // Convert string to hexutil.Big
		Hash:             common.HexToHash(tx.Hash),                          // Convert string to common.Hash
		Input:            hexutil.Bytes(tx.Input),                            // Convert string to hexutil.Bytes
		Nonce:            hexutil.Uint64(hexutil.MustDecodeUint64(tx.Nonce)), // Convert string to hexutil.Uint64
		To:               &to,                                                // Convert string to common.Address
		TransactionIndex: &txIndex,                                           // Convert string to hexutil.Uint64
		Value:            &value,                                             // Convert string to hexutil.Big
		Type:             hexutil.Uint64(hexutil.MustDecodeUint64(tx.Type)),  // Convert string to hexutil.Uint64
		Accesses:         &ethtypes.AccessList{},                             // Assuming you handle this appropriately
		ChainID:          &chainId,                                           // Convert string to hexutil.Big
		V:                &v,                                                 // Convert string to hexutil.Big
		R:                &r,                                                 // Convert string to hexutil.Big
		S:                &s,                                                 // Convert string
	}
}

// GetAccountTransactionsByPageAndSize returns the transactions at the given page and size for the filtered by address
func (b *Backend) GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	if page == 0 || size == 0 || size > 100 {
		return nil, errors.New("invalid page or size parameters")
	}

	// Transactions accumulator
	transactions := make([]*rpctypes.RPCTransaction, 0, size)

	// Check for Chronos wallet transactions first
	if _, err := b.GetCronByAddress(address); err == nil {
		chronTxs, err := b.GetCronTransactionsByPageAndSize(address, page, size)
		if err != nil {
			return nil, err
		}
		for _, tx := range chronTxs {
			transactions = append(transactions, convertChronosRPCTransactionToRPCTransaction(tx))
		}
		return transactions, nil
	}

	// utilities
	counter := uint64(0)
	skip := (uint64(page) - 1) * uint64(size)

	// Handle pending transactions first
	pendingTxs, _ := b.PendingTransactions()
	for _, tx := range pendingTxs {
		msg, err := evmtypes.UnwrapEthereumMsgTx(tx)
		if err != nil {
			continue
		}
		rpctx, err := rpctypes.NewTransactionFromMsg(msg, common.Hash{}, 0, 0, nil, b.chainID)
		if err != nil {
			continue
		}
		if rpctx.From == address || (rpctx.To != nil && *rpctx.To == address) {
			if counter >= skip {
				transactions = append(transactions, rpctx)
				if uint64(len(transactions)) >= uint64(size) {
					return transactions, nil
				}
			}
			counter++
		}
	}

	// Fetch relevant blocks heights once
	heightsWhereFindingTx := make([]int64, 0)
	batchSize := int64(1000)
	latestBlockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, err
	}
	maxHeight := int64(latestBlockNumber)
	totalMatchedTxs := uint64(0)

	for maxHeight > 0 && totalMatchedTxs < skip+uint64(size) {
		minHeight := maxHeight - batchSize + 1
		if minHeight < 1 {
			minHeight = 1
		}

		info, err := b.clientCtx.Client.BlockchainLocateTxsInfo(b.ctx, minHeight, maxHeight, address.String(), "")
		if err != nil {
			return nil, fmt.Errorf("failed to get block metadata: %w", err)
		}

		for _, blockMeta := range info.BlockMetas {
			if blockMeta.NumTxs > 0 {
				heightsWhereFindingTx = append(heightsWhereFindingTx, blockMeta.Header.Height)
				totalMatchedTxs += uint64(blockMeta.NumTxs) // Compteur explicite ici
			}
		}
		maxHeight = minHeight - 1
		if minHeight == 1 {
			break
		}
	}

	// Reverse iterate relevant heights to maintain asc order
	for i := 0; i < len(heightsWhereFindingTx) && uint64(len(transactions)) < uint64(size); i++ {
		blockHeight := heightsWhereFindingTx[i]
		block, err := b.GetBlockByNumber(rpctypes.BlockNumber(blockHeight), true)
		if err != nil {
			continue
		}
		txs, ok := block["transactions"].([]interface{})
		if !ok {
			continue
		}
		for _, tx := range txs {
			ethTx, ok := tx.(*rpctypes.RPCTransaction)
			if !ok {
				continue
			}
			if ethTx.From == address || (ethTx.To != nil && *ethTx.To == address) {
				if counter >= skip {
					transactions = append(transactions, ethTx)
					if uint64(len(transactions)) >= uint64(size) {
						return transactions, nil
					}
				}
				counter++
			}
		}
	}

	return transactions, nil
}
