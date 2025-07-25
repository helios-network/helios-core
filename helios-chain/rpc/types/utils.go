package types

import (
	"context"
	_ "embed"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"bytes"
	"encoding/base64"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	tmrpcclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cosmos/cosmos-sdk/client"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	"helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// ExceedBlockGasLimitError defines the error message when tx execution exceeds the block gas limit.
// The tx fee is deducted in ante handler, so it shouldn't be ignored in JSON-RPC API.
const ExceedBlockGasLimitError = "out of gas in location: block gas meter; gasWanted:"

// StateDBCommitError defines the error message when commit after executing EVM transaction, for example
// transfer native token to a distribution module account 0x93354845030274cD4bf1686Abd60AB28EC52e1a7 using an evm type transaction
// note: the transfer amount cannot be set to 0, otherwise this problem will not be triggered
const StateDBCommitError = "failed to commit stateDB"

// RawTxToEthTx returns a evm MsgEthereum transaction from raw tx bytes.
func RawTxToEthTx(clientCtx client.Context, txBz cmttypes.Tx) ([]*evmtypes.MsgEthereumTx, error) {
	tx, err := clientCtx.TxConfig.TxDecoder()(txBz)
	if err != nil {
		return nil, errorsmod.Wrap(errortypes.ErrJSONUnmarshal, err.Error())
	}

	ethTxs := make([]*evmtypes.MsgEthereumTx, len(tx.GetMsgs()))
	for i, msg := range tx.GetMsgs() {
		ethTx, ok := msg.(*evmtypes.MsgEthereumTx)
		if !ok {
			return nil, fmt.Errorf("invalid message type %T, expected %T", msg, &evmtypes.MsgEthereumTx{})
		}
		ethTx.Hash = ethTx.AsTransaction().Hash().Hex()
		ethTxs[i] = ethTx
	}
	return ethTxs, nil
}

// EthHeaderFromTendermint is an util function that returns an Ethereum Header
// from a tendermint Header.
func EthHeaderFromTendermint(header cmttypes.Header, bloom ethtypes.Bloom, baseFee *big.Int) *ethtypes.Header {
	txHash := ethtypes.EmptyRootHash
	if len(header.DataHash) == 0 {
		txHash = common.BytesToHash(header.DataHash)
	}

	time := uint64(header.Time.UTC().Unix()) // #nosec G701 G115
	return &ethtypes.Header{
		ParentHash:  common.BytesToHash(header.LastBlockID.Hash.Bytes()),
		UncleHash:   ethtypes.EmptyUncleHash,
		Coinbase:    common.BytesToAddress(header.ProposerAddress),
		Root:        common.BytesToHash(header.AppHash),
		TxHash:      txHash,
		ReceiptHash: ethtypes.EmptyRootHash,
		Bloom:       bloom,
		Difficulty:  big.NewInt(0),
		Number:      big.NewInt(header.Height),
		GasLimit:    0,
		GasUsed:     0,
		Time:        time,
		Extra:       []byte{},
		MixDigest:   common.Hash{},
		Nonce:       ethtypes.BlockNonce{},
		BaseFee:     baseFee,
	}
}

// BlockMaxGasFromConsensusParams returns the gas limit for the current block from the chain consensus params.
func BlockMaxGasFromConsensusParams(goCtx context.Context, clientCtx client.Context, blockHeight int64) (int64, error) {
	tmrpcClient, ok := clientCtx.Client.(tmrpcclient.Client)
	if !ok {
		panic("incorrect tm rpc client")
	}
	resConsParams, err := tmrpcClient.ConsensusParams(goCtx, &blockHeight)
	defaultGasLimit := int64(^uint32(0)) // #nosec G701
	if err != nil {
		return defaultGasLimit, err
	}

	gasLimit := resConsParams.ConsensusParams.Block.MaxGas
	if gasLimit == -1 {
		// Sets gas limit to max uint32 to not error with javascript dev tooling
		// This -1 value indicating no block gas limit is set to max uint64 with geth hexutils
		// which errors certain javascript dev tooling which only supports up to 53 bits
		gasLimit = defaultGasLimit
	}

	return gasLimit, nil
}

// FormatBlock creates an ethereum block from a tendermint header and ethereum-formatted
// transactions.
func FormatBlock(
	header cmttypes.Header, size int, gasLimit int64,
	gasUsed *big.Int, transactions []interface{}, bloom ethtypes.Bloom,
	validatorAddr common.Address, baseFee *big.Int,
	cronTransactions []interface{},
) map[string]interface{} {
	var transactionsRoot common.Hash
	if len(transactions) == 0 {
		transactionsRoot = ethtypes.EmptyRootHash
	} else {
		transactionsRoot = common.BytesToHash(header.DataHash)
	}

	result := map[string]interface{}{
		"number":           hexutil.Uint64(header.Height), //nolint:gosec // G115
		"hash":             hexutil.Bytes(header.Hash()),
		"parentHash":       common.BytesToHash(header.LastBlockID.Hash.Bytes()),
		"nonce":            ethtypes.BlockNonce{},   // PoW specific
		"sha3Uncles":       ethtypes.EmptyUncleHash, // No uncles in Tendermint
		"logsBloom":        bloom,
		"stateRoot":        hexutil.Bytes(header.AppHash),
		"miner":            validatorAddr,
		"mixHash":          common.Hash{},
		"difficulty":       (*hexutil.Big)(big.NewInt(0)),
		"extraData":        "0x",
		"size":             hexutil.Uint64(size),     //nolint:gosec // G115
		"gasLimit":         hexutil.Uint64(gasLimit), //nolint:gosec // G115 -- Static gas limit
		"gasUsed":          (*hexutil.Big)(gasUsed),
		"timestamp":        hexutil.Uint64(header.Time.Unix()), //nolint:gosec // G115
		"transactionsRoot": transactionsRoot,
		"receiptsRoot":     ethtypes.EmptyRootHash,

		"uncles":           []common.Hash{},
		"transactions":     transactions,
		"cronTransactions": cronTransactions,
		"totalDifficulty":  (*hexutil.Big)(big.NewInt(0)),
	}

	if baseFee != nil {
		result["baseFeePerGas"] = (*hexutil.Big)(baseFee)
	}

	return result
}

// NewTransactionFromMsg returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewTransactionFromMsg(
	msg *evmtypes.MsgEthereumTx,
	blockHash common.Hash,
	blockNumber, index uint64,
	baseFee *big.Int,
	chainID *big.Int,
) (*RPCTransaction, error) {
	tx := msg.AsTransaction()
	return NewRPCTransaction(tx, blockHash, blockNumber, index, baseFee, chainID)
}

func NewUnsignedTransactionFromMsg(
	msg *evmtypes.MsgEthereumTx,
	blockHash common.Hash,
	blockNumber, index uint64,
	baseFee *big.Int,
	chainID *big.Int,
	from common.Address,
) (*RPCTransaction, error) {
	tx := msg.AsTransaction()
	return NewRPCUnsignedTransaction(tx, blockHash, blockNumber, index, baseFee, chainID, from)
}

// Helper pour parser un uint64 depuis une string (retourne 0 si vide ou erreur)
func parseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// Helper pour parser un *big.Int depuis une string (retourne 0 si vide ou erreur)
func parseBigInt(s string) *big.Int {
	if s == "" {
		return big.NewInt(0)
	}
	val, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return val
}

// Helper pour parser un hash (hex string)
func parseHash(s string) common.Hash {
	if s == "" {
		return common.Hash{}
	}
	return common.HexToHash(s)
}

// Helper pour parser une address (hex string)
func parseAddress(s string) common.Address {
	if s == "" {
		return common.Address{}
	}
	return common.HexToAddress(s)
}

// Helper pour parser les input data (hex string)
func parseInput(s string) hexutil.Bytes {
	if s == "" {
		return nil
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil
	}
	return b
}

func NewRPCTransactionFromCronTransaction(
	tx *types.CronTransactionRPC,
) (*RPCTransaction, error) {
	result := &RPCTransaction{
		Type:     hexutil.Uint64(hexutil.MustDecodeUint64(tx.Type)),
		From:     parseAddress(tx.From),
		Gas:      hexutil.Uint64(hexutil.MustDecodeUint64(tx.Gas)),
		GasPrice: (*hexutil.Big)(hexutil.MustDecodeBig(tx.GasPrice)),
		Hash:     parseHash(tx.Hash),
		Input:    parseInput(tx.Input),
		Nonce:    hexutil.Uint64(hexutil.MustDecodeUint64(tx.Nonce)),
		To:       nil,
		Value:    (*hexutil.Big)(hexutil.MustDecodeBig(tx.Value)),
		V:        (*hexutil.Big)(hexutil.MustDecodeBig(tx.V)),
		R:        (*hexutil.Big)(hexutil.MustDecodeBig(tx.R)),
		S:        (*hexutil.Big)(hexutil.MustDecodeBig(tx.S)),
		ChainID:  (*hexutil.Big)(hexutil.MustDecodeBig(tx.ChainId)),
	}
	if tx.To != "" {
		addr := parseAddress(tx.To)
		result.To = &addr
	}
	// Optionnel: TransactionIndex, BlockHash, BlockNumber...
	if tx.TransactionIndex != "" {
		idx := hexutil.MustDecodeUint64(tx.TransactionIndex)
		result.TransactionIndex = (*hexutil.Uint64)(&idx)
	}
	if tx.BlockHash != "" {
		bh := parseHash(tx.BlockHash)
		result.BlockHash = &bh
	}
	if tx.BlockNumber != "" {
		bn := hexutil.MustDecodeUint64(tx.BlockNumber)
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(bn))
	}
	return result, nil
}

// NewTransactionFromData returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func NewRPCTransaction(
	tx *ethtypes.Transaction,
	blockHash common.Hash,
	blockNumber,
	index uint64,
	baseFee,
	chainID *big.Int,
) (*RPCTransaction, error) {
	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	var signer ethtypes.Signer
	if tx.Protected() {
		signer = ethtypes.LatestSignerForChainID(tx.ChainId())
	} else {
		signer = ethtypes.HomesteadSigner{}
	}
	from, _ := ethtypes.Sender(signer, tx) // #nosec G703
	v, r, s := tx.RawSignatureValues()
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
		ChainID:  (*hexutil.Big)(chainID),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	switch tx.Type() {
	case ethtypes.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case ethtypes.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}

	return result, nil
}

func NewRPCUnsignedTransaction(
	tx *ethtypes.Transaction,
	blockHash common.Hash,
	blockNumber,
	index uint64,
	baseFee,
	chainID *big.Int,
	from common.Address,
) (*RPCTransaction, error) {
	// Determine the signer. For replay-protected transactions, use the most permissive
	// signer, because we assume that signers are backwards-compatible with old
	// transactions. For non-protected transactions, the homestead signer signer is used
	// because the return value of ChainId is zero for those transactions.
	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    hexutil.Bytes(tx.Data()),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(big.NewInt(0)),
		R:        (*hexutil.Big)(big.NewInt(0)),
		S:        (*hexutil.Big)(big.NewInt(0)),
		ChainID:  (*hexutil.Big)(chainID),
	}
	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(new(big.Int).SetUint64(blockNumber))
		result.TransactionIndex = (*hexutil.Uint64)(&index)
	}
	switch tx.Type() {
	case ethtypes.AccessListTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	case ethtypes.DynamicFeeTxType:
		al := tx.AccessList()
		result.Accesses = &al
		result.ChainID = (*hexutil.Big)(tx.ChainId())
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		// if the transaction has been mined, compute the effective gas price
		if baseFee != nil && blockHash != (common.Hash{}) {
			// price = min(tip, gasFeeCap - baseFee) + baseFee
			price := math.BigMin(new(big.Int).Add(tx.GasTipCap(), baseFee), tx.GasFeeCap())
			result.GasPrice = (*hexutil.Big)(price)
		} else {
			result.GasPrice = (*hexutil.Big)(tx.GasFeeCap())
		}
	}

	return result, nil
}

// BaseFeeFromEvents parses the feemarket basefee from cosmos events
func BaseFeeFromEvents(events []abci.Event) *big.Int {
	for _, event := range events {
		if event.Type != evmtypes.EventTypeFeeMarket {
			continue
		}

		for _, attr := range event.Attributes {
			if attr.Key == evmtypes.AttributeKeyBaseFee {
				result, success := sdkmath.NewIntFromString(attr.Value)
				if success {
					return result.BigInt()
				}

				return nil
			}
		}
	}
	return nil
}

// CheckTxFee is an internal function used to check whether the fee of
// the given transaction is _reasonable_(under the minimum cap).
func CheckTxFee(gasPrice *big.Int, gas uint64, minCap float64) error {
	// Short circuit if there is no cap for transaction fee at all.
	if minCap == 0 {
		return nil
	}
	totalfee := new(big.Float).SetInt(new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gas)))
	// 1 evmos in 10^18 aevmos
	oneToken := new(big.Float).SetInt(big.NewInt(params.Ether))
	// quo = rounded(x/y)
	feeEth := new(big.Float).Quo(totalfee, oneToken)
	// no need to check error from parsing
	feeFloat, _ := feeEth.Float64()
	if feeFloat > minCap {
		return fmt.Errorf("tx fee (%.2f ether) exceeds the configured cap (%.2f ether)", feeFloat, minCap)
	}
	return nil
}

// TxExceedBlockGasLimit returns true if the tx exceeds block gas limit.
func TxExceedBlockGasLimit(res *abci.ExecTxResult) bool {
	return strings.Contains(res.Log, ExceedBlockGasLimitError)
}

// TxStateDBCommitError returns true if the evm tx commit error.
func TxStateDBCommitError(res *abci.ExecTxResult) bool {
	return strings.Contains(res.Log, StateDBCommitError)
}

// TxSucessOrExpectedFailure returns true if the transaction was successful
// or if it failed with an ExceedBlockGasLimit error or TxStateDBCommitError error
func TxSucessOrExpectedFailure(res *abci.ExecTxResult) bool {
	return res.Code == 0 || TxExceedBlockGasLimit(res) || TxStateDBCommitError(res)
}

func GenerateTokenLogoBase64(symbol string) (string, error) {
	const finalSize = 200
	const charHeight = 13
	const margin = 4

	// Setup dummy drawer to measure string
	face := basicfont.Face7x13
	dummyImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
	d := &font.Drawer{
		Dst:  dummyImg,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	textWidth := d.MeasureString(symbol).Round()
	textHeight := charHeight

	width := textWidth + 2*margin
	height := textHeight + 2*margin

	imgSize := width
	if height > width {
		imgSize = height
	}

	// Create image
	baseImg := image.NewRGBA(image.Rect(0, 0, imgSize, imgSize))
	draw.Draw(baseImg, baseImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	// Draw actual text
	d = &font.Drawer{
		Dst:  baseImg,
		Src:  image.NewUniform(color.Black),
		Face: face,
	}
	x := (imgSize - textWidth) / 2
	y := (imgSize + charHeight) / 2
	d.Dot = fixed.P(x, y)
	d.DrawString(symbol)

	// Scale to 200x200
	finalImg := image.NewRGBA(image.Rect(0, 0, finalSize, finalSize))
	draw.NearestNeighbor.Scale(finalImg, finalImg.Bounds(), baseImg, baseImg.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	if err := png.Encode(&buf, finalImg); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
