package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	_ TxData = &LegacyTx{}
	_ TxData = &AccessListTx{}
	_ TxData = &DynamicFeeTx{}
)

// TxData implements the Ethereum transaction tx structure. It is used
// solely as intended in Ethereum abiding by the protocol.
type TxData interface {
	// TODO: embed ethtypes.TxData. See https://github.com/ethereum/go-ethereum/issues/23154

	TxType() byte
	Copy() TxData
	GetChainID() *big.Int
	GetAccessList() ethtypes.AccessList
	GetData() []byte
	GetNonce() uint64
	GetGas() uint64
	GetGasPrice() *big.Int
	GetGasTipCap() *big.Int
	GetGasFeeCap() *big.Int
	GetValue() *big.Int
	GetTo() *common.Address

	GetRawSignatureValues() (v, r, s *big.Int)
	SetSignatureValues(chainID, v, r, s *big.Int)

	AsEthereumData() ethtypes.TxData
	Validate() error

	// Fee returns the maximum fee a sender of a message is willing to pay.
	Fee() *big.Int
	// Cost returns the total cost of a transaction before executing any smart
	// contract call. This means it should return the fee the user has to pay
	// plus the amount of tokens they want to transfer.
	Cost() *big.Int

	// EffectiveGasPrice returns the price for the gas used in a transaction
	// based on the transaction type.
	EffectiveGasPrice(baseFee *big.Int) *big.Int
	// EffectiveFee returns the fees a user is willing to pay for a transaction.
	EffectiveFee(baseFee *big.Int) *big.Int
	EffectiveCost(baseFee *big.Int) *big.Int
}

// NOTE: All non-protected transactions (i.e non EIP155 signed) will fail if the
// AllowUnprotectedTxs parameter is disabled.
func NewTxDataFromTx(tx *ethtypes.Transaction) (TxData, error) {
	var txData TxData
	var err error
	switch tx.Type() {
	case ethtypes.DynamicFeeTxType:
		txData, err = NewDynamicFeeTx(tx)
	case ethtypes.AccessListTxType:
		txData, err = newAccessListTx(tx)
	default:
		txData, err = NewLegacyTx(tx)
	}
	if err != nil {
		return nil, err
	}

	return txData, nil
}

// fee returns the fee for a transaction given by the gas price time the gas.
func fee(gasPrice *big.Int, gas uint64) *big.Int {
	gasLimit := new(big.Int).SetUint64(gas)
	return new(big.Int).Mul(gasPrice, gasLimit)
}

// cost returns the sum of the fee and value. If value is nil it returns only
// the fee. This function is made to be used in the ante handler to compute the
// total cost of a transaction given by the fee the user has to pay and the
// amount they want to transfer.
func cost(fee, value *big.Int) *big.Int {
	if value != nil {
		return new(big.Int).Add(fee, value)
	}
	return fee
}
