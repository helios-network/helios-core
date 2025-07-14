package types

import (
	"fmt"
	"math/big"
	"strings"

	accountsabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// GetCheckpoint gets the checkpoint signature from the given outgoing tx batch
func (b OutgoingTxBatch) GetCheckpoint(hyperionID uint64) common.Hash {

	abi, err := accountsabi.JSON(strings.NewReader(OutgoingBatchTxCheckpointABIJSON))
	if err != nil {
		panic("Bad ABI constant!")
	}

	// Create the methodName argument which salts the signature
	methodNameBytes := []uint8("transactionBatch")
	var batchMethodName [32]uint8
	copy(batchMethodName[:], methodNameBytes)

	// Run through the elements of the batch and serialize them
	txAmounts := make([]*big.Int, len(b.Transactions))
	txDestinations := make([]common.Address, len(b.Transactions))
	txFees := make([]*big.Int, len(b.Transactions))
	for i, tx := range b.Transactions {
		txAmounts[i] = tx.Token.Amount.BigInt()
		txDestinations[i] = common.HexToAddress(tx.DestAddress)
		txFees[i] = big.NewInt(0) //tx.Token.Amount.BigInt()
	}

	hyperionIDBytes := []uint8("checkpoint")
	var tHyperionID [32]uint8
	copy(tHyperionID[:], hyperionIDBytes)

	// the methodName needs to be the same as the 'name' above in the checkpointAbiJson
	// but other than that it's a constant that has no impact on the output. This is because
	// it gets encoded as a function name which we must then discard.
	// TODO: hyperionID is bytes32 NOT uint64
	abiEncodedBatch, err := abi.Pack("submitBatch",
		tHyperionID,
		batchMethodName,
		txAmounts,
		txDestinations,
		txFees,
		big.NewInt(int64(b.BatchNonce)),
		common.HexToAddress(b.TokenContract),
		big.NewInt(int64(b.BatchTimeout)),
	)

	// this should never happen outside of test since any case that could crash on encoding
	// should be filtered above.
	if err != nil {
		panic(fmt.Sprintf("Error packing checkpoint! %s/n", err))
	}

	hash := crypto.Keccak256Hash(abiEncodedBatch[4:])
	return hash
}
