package types

// import (
// 	evmtypes "helios-core/helios-chain/x/evm/types"

// 	abcitypes "github.com/cometbft/cometbft/abci/types"
// )

// type ScheduleTxResult struct {
// 	Tx     evmtypes.MsgEthereumTx `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
// 	Result abcitypes.ExecTxResult `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
// }

const (
	// BloomByteLength represents the number of bytes used in a header log bloom.
	BloomByteLength = 256

	// BloomBitLength represents the number of bits used in a header log bloom.
	BloomBitLength = 8 * BloomByteLength
)

// Bloom represents a 2048 bit bloom filter.
type Bloom [BloomByteLength]byte
