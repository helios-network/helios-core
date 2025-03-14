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

// GetAccountTransactionsByPageAndSize returns the transactions at the given page and size for the filtered by address
func (b *Backend) GetAccountTransactionsByPageAndSize(address common.Address, page hexutil.Uint64, size hexutil.Uint64) ([]*rpctypes.RPCTransaction, error) {
	if page == 0 || size == 0 {
		return nil, errors.New("page and size must be greater than 0")
	}

	if size > 100 {
		return nil, errors.New("size must be less than 100")
	}

	sizeNum := uint64(size)
	pageNum := uint64(page)

	transactions := make([]*rpctypes.RPCTransaction, 0, sizeNum)

	pendingTxs, err := b.PendingTransactions()
	if err != nil {
		return nil, err
	}

	pageOffset := (pageNum - 1) * sizeNum

	pendingTxCount := uint64(len(pendingTxs))

	// add mempool txs if we are in the page range
	if pageOffset < pendingTxCount {
		pendingEndPosition := min(pageOffset+sizeNum, pendingTxCount)

		for i := pageOffset; i < pendingEndPosition; i++ {
			msg, err := evmtypes.UnwrapEthereumMsgTx(pendingTxs[i])
			if err != nil {
				return nil, fmt.Errorf("failed to unwrap ethereum tx: %w", err)
			}

			if !isAddressInvolved(msg, address) {
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
				return nil, fmt.Errorf("failed to create transaction from msg: %w", err)
			}

			transactions = append(transactions, rpctx)

			if uint64(len(transactions)) >= sizeNum {
				return transactions, nil
			}
		}
	}

	// we need to find txs in confirmed blocks
	latestBlockNumber, err := b.BlockNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block number: %w", err)
	}

	// calculate the updated page offset
	updatedPageOffset := pageOffset - uint64(len(transactions))

	// use BlockchainInfo to estimate the target block based on num_txs using batches of 100
	const batchSize = int64(100)
	maxHeight := int64(latestBlockNumber)
	var targetBlock int64 = -1
	txCount := uint64(0)

	// find target block to start searching for txs and filter by address
	for maxHeight > 0 {
		minHeight := int64(1)
		if maxHeight > int64(batchSize) {
			minHeight = maxHeight - int64(batchSize) + 1
		}

		info, err := b.clientCtx.Client.BlockchainInfo(b.ctx, int64(minHeight), int64(maxHeight))
		if err != nil {
			return nil, fmt.Errorf("failed to get block metadata: %w", err)
		}
		for i := 0; i < len(info.BlockMetas); i++ {
			txCount += uint64(info.BlockMetas[i].NumTxs)
			if txCount >= updatedPageOffset {
				targetBlock = info.BlockMetas[i].Header.Height
				break
			}
		}
		if targetBlock != -1 {
			break
		}
		if maxHeight > int64(batchSize) {
			maxHeight -= int64(batchSize)
		} else {
			break
		}
	}

	if targetBlock == -1 {
		return nil, fmt.Errorf("txs not found for page %d and size %d", pageNum, sizeNum)
	}

	// search for txs filtered by address starting from the target block in reverse order
	for blockHeight := targetBlock; blockHeight > 0 && uint64(len(transactions)) < sizeNum; blockHeight-- {

		block, err := b.rpcClient.Block(b.ctx, &blockHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to get block %d: %w", blockHeight, err)
		}

		blockRes, err := b.rpcClient.BlockResults(b.ctx, &blockHeight)
		if err != nil {
			return nil, fmt.Errorf("failed to get block results for block %d: %w", blockHeight, err)
		}

		// pre filter messages by address before creating full RPC tx objects
		ethMsgs := b.EthMsgsFromTendermintBlock(block, blockRes)
		filteredMsgs := make([]*evmtypes.MsgEthereumTx, 0)
		for _, msg := range ethMsgs {
			if isAddressInvolved(msg, address) {
				filteredMsgs = append(filteredMsgs, msg)
			}
		}

		// only fetch base fee if we have txs filtered by address
		if len(filteredMsgs) > 0 {
			baseFee, err := b.BaseFee(blockRes)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch Base Fee from prunned block. Check node prunning configuration: %w", err)
			}

			// now create full RPC tx objects
			for i, msg := range filteredMsgs {
				rpctx, err := rpctypes.NewTransactionFromMsg(
					msg,
					common.BytesToHash(block.Block.Hash()),
					uint64(block.Block.Height),
					uint64(i),
					baseFee,
					b.chainID,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create transaction from msg: %w", err)
				}
				transactions = append(transactions, rpctx)
				if uint64(len(transactions)) >= sizeNum {
					break
				}
			}
		}
	}
	return transactions, nil
}

// isAddressInvolved is a helper method that checks if an address is involved in a transaction as sender or receiver
func isAddressInvolved(msg *evmtypes.MsgEthereumTx, address common.Address) bool {
	// Check sender (From)
	if common.HexToAddress(msg.From) == address {
		return true
	}

	// Unpack the transaction data
	txData, err := evmtypes.UnpackTxData(msg.Data)
	if err != nil {
		return false
	}

	// Different transaction types have different ways to access the recipient
	switch txData := txData.(type) {
	case *evmtypes.LegacyTx:
		if txData.To != "" && common.HexToAddress(txData.To) == address {
			return true
		}
	case *evmtypes.AccessListTx:
		if txData.To != "" && common.HexToAddress(txData.To) == address {
			return true
		}
	case *evmtypes.DynamicFeeTx:
		if txData.To != "" && common.HexToAddress(txData.To) == address {
			return true
		}
	}

	return false
}
