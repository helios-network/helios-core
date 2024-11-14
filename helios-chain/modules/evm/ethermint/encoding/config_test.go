package encoding_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"helios-core/helios-chain/modules/evm/ethermint/encoding"
	"helios-core/helios-chain/modules/evm/ethermint/tests"
	evmtypes "helios-core/helios-chain/modules/evm/evm/types"
)

func TestTxEncoding(t *testing.T) {
	addr, key := tests.NewAddrKey()
	signer := tests.NewSigner(key)

	msg := evmtypes.NewTxContract(big.NewInt(1), 1, big.NewInt(10), 100000, nil, big.NewInt(1), big.NewInt(1), []byte{}, nil)
	msg.From = addr.Bytes()

	ethSigner := ethtypes.LatestSignerForChainID(big.NewInt(1))
	err := msg.Sign(ethSigner, signer)
	require.NoError(t, err)
	_, err = encoding.MakeConfig().TxConfig.TxEncoder()(msg)
	require.Error(t, err, "encoding failed")

	// FIXME: transaction hashing is hardcoded on Terndermint:
	// See https://github.com/tendermint/tendermint/issues/6539 for reference
	// txHash := msg.AsTransaction().Hash()
	// tmTx := tmtypes.Tx(bz)

	// require.Equal(t, txHash.Bytes(), tmTx.Hash())
}
