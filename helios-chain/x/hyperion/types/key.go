package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	// ModuleName is the name of the module
	ModuleName = "hyperion"

	// StoreKey to be used when creating the KVStore
	StoreKey = ModuleName

	// RouterKey is the module name router key
	RouterKey = ModuleName

	// QuerierRoute to be used for querierer msgs
	QuerierRoute = ModuleName
)

var (
	// EthAddressByValidatorKey indexes cosmos validator account addresses
	// i.e. cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn
	EthAddressByValidatorKey = []byte{0x1}

	// ValidatorByEthAddressKey indexes ethereum addresses
	// i.e. 0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B
	ValidatorByEthAddressKey = []byte{0xfb}

	// ValsetRequestKey indexes valset requests by nonce
	ValsetRequestKey = []byte{0x2}

	// ValsetConfirmKey indexes valset confirmations by nonce and the validator account address
	// i.e cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn
	ValsetConfirmKey = []byte{0x3}

	// ParamKey indexes hyperion's module params
	ParamKey = []byte{0x4}

	// OracleAttestationKey attestation details by nonce and validator address
	// i.e. cosmosvaloper1ahx7f8wyertuus9r20284ej0asrs085case3kn
	// An attestation can be thought of as the 'event to be executed' while
	// the Claims are an individual validator saying that they saw an event
	// occur the Attestation is 'the event' that multiple claims vote on and
	// eventually executes
	OracleAttestationKey = []byte{0x5}

	// OutgoingTXPoolKey indexes the last nonce for the outgoing tx pool
	OutgoingTXPoolKey = []byte{0x6}

	// SecondIndexOutgoingTXFeeKey indexes fee amounts by token contract address
	SecondIndexOutgoingTXFeeKey = []byte{0x9}

	// OutgoingTXBatchKey indexes outgoing tx batches under a nonce and token address
	OutgoingTXBatchKey = []byte{0xa}

	// OutgoingTXBatchBlockKey indexes outgoing tx batches under a block height and token address
	OutgoingTXBatchBlockKey = []byte{0xb}

	// BatchConfirmKey indexes validator confirmations by token contract address
	BatchConfirmKey = []byte{0xe1}

	// LastEventNonceByValidatorKey indexes lateset event nonce by validator
	LastEventNonceByValidatorKey = []byte{0xe2}

	// LastEventByValidatorKey indexes lateset claim event by validator
	LastEventByValidatorKey = []byte{0xf1}

	// LastObservedEventNonceKey indexes the latest event nonce
	LastObservedEventNonceKey = []byte{0xf2}

	// SequenceKeyPrefix indexes different txids
	SequenceKeyPrefix = []byte{0x7}

	// KeyLastTXPoolID indexes the lastTxPoolID
	KeyLastTXPoolID = append(SequenceKeyPrefix, []byte("lastTxPoolId")...)

	// KeyLastOutgoingBatchID indexes the lastBatchID
	KeyLastOutgoingBatchID = append(SequenceKeyPrefix, []byte("lastBatchId")...)

	// KeyOrchestratorAddress indexes the validator keys for an orchestrator
	KeyOrchestratorAddress = []byte{0xe8}

	// LastObservedEthereumBlockHeightKey indexes the latest Ethereum block height
	LastObservedEthereumBlockHeightKey = []byte{0xf9}

	// DenomToTokenAddressKey prefixes the index of Cosmos originated asset denoms to ERC20s
	DenomToTokenAddressKey = []byte{0xf3}

	// TokenAddressToDenomKey prefixes the index of Cosmos originated asset denoms to ERC20s
	TokenAddressToDenomKey = []byte{0xf4}

	// LastSlashedValsetNonce indexes the latest slashed valset nonce
	LastSlashedValsetNonce = []byte{0xf5}

	// LatestValsetNonce indexes the latest valset nonce
	LatestValsetNonce = []byte{0xf6}

	// LastSlashedBatchBlock indexes the latest slashed batch block height
	LastSlashedBatchBlock = []byte{0xf7}

	// LastUnbondingBlockHeight indexes the last validator unbonding block height
	LastUnbondingBlockHeight = []byte{0xf8}

	// LastObservedValsetNonceKey indexes the latest observed valset nonce
	// HERE THERE BE DRAGONS, do not use this value as an up to date validator set
	// on Ethereum it will always lag significantly and may be totally wrong at some
	// times.
	LastObservedValsetKey = []byte{0xfa}

	// PastEthSignatureCheckpointKey indexes eth signature checkpoints that have existed
	PastEthSignatureCheckpointKey = []byte{0x1b}

	EthereumBlacklistKey = []byte{0x1c}

	FinalizedTxKey = []byte{0x1d}
)

func GetEthereumBlacklistStoreKey(addr common.Address) []byte {
	return append(EthereumBlacklistKey, addr.Bytes()...)
}

// GetOrchestratorAddressKey returns the following key format
// prefix
// [0xe8][cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn]
func GetOrchestratorAddressKey(hyperionId uint64, orc sdk.AccAddress) []byte {
	buf := make([]byte, 0, len(KeyOrchestratorAddress)+8+len(orc.Bytes()))
	buf = append(buf, KeyOrchestratorAddress...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, orc.Bytes()...)
	return buf
}

// GetEthAddressByValidatorKey returns the following key format
// prefix      hyperionId        cosmos-validator
// [0x0][0 0 0 0 0 0 0 1][cosmosvaloper1ahx7f8wyertuus9r20284ej0asrs085case3kn]
func GetEthAddressByValidatorKey(hyperionId uint64, validator sdk.ValAddress) []byte {
	buf := make([]byte, 0, len(EthAddressByValidatorKey)+8+len(validator.Bytes()))
	buf = append(buf, EthAddressByValidatorKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, validator.Bytes()...)
	return buf
}

// GetValidatorByEthAddressKey returns the following key format
// prefix      hyperionId        cosmos-validator
// [0xf9][0 0 0 0 0 0 0 1][0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B]
func GetValidatorByEthAddressKey(hyperionId uint64, ethAddress common.Address) []byte {
	buf := make([]byte, 0, len(ValidatorByEthAddressKey)+8+len(ethAddress.Bytes()))
	buf = append(buf, ValidatorByEthAddressKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, ethAddress.Bytes()...)
	return buf
}

// GetValsetKey returns the following key format
// prefix   hyperionId      nonce
// [0x0][0 0 0 0 0 0 0 1][0 0 0 0 0 0 0 1]
func GetValsetKey(hyperionId uint64, nonce uint64) []byte {
	buf := make([]byte, 0, len(ValsetRequestKey)+8+8)
	buf = append(buf, ValsetRequestKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, UInt64Bytes(nonce)...)

	return buf
}

func GetLatestValsetKey(hyperionId uint64) []byte {
	return append(LatestValsetNonce, UInt64Bytes(hyperionId)...)
}

func GetLastSlashedBatchBlockKey(hyperionId uint64) []byte {
	return append(LastSlashedBatchBlock, UInt64Bytes(hyperionId)...)
}

// GetValsetConfirmKey returns the following key format
// prefix    hyperionId       nonce                    validator-address
// [0x0][0 0 0 0 0 0 0 1][0 0 0 0 0 0 0 1][cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn]
// MARK finish-batches: this is where the key is created in the old (presumed working) code
func GetValsetConfirmKey(hyperionId uint64, nonce uint64, validator sdk.AccAddress) []byte {
	buf := make([]byte, 0, len(ValsetConfirmKey)+8+8+len(validator))
	buf = append(buf, ValsetConfirmKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, UInt64Bytes(nonce)...)
	buf = append(buf, validator.Bytes()...)

	return buf
}

// for iterate on ValsetConfirmKeys
func GetValsetConfirmPrefixKey(hyperionId uint64, nonce uint64) []byte {
	buf := make([]byte, 0, 8+8)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, UInt64Bytes(nonce)...)

	return buf
}

func GetLastUnbondingBlockHeightKey() []byte {
	buf := make([]byte, 0, len(LastUnbondingBlockHeight))
	buf = append(buf, LastUnbondingBlockHeight...)
	return buf
}

func GetLastSlashedValsetNonceKey(hyperionId uint64) []byte {
	buf := make([]byte, 0, len(LastSlashedValsetNonce)+8)
	buf = append(buf, LastSlashedValsetNonce...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	return buf
}

// GetAttestationKeyWithHash returns the following key format
// prefix     nonce                             claim-details-hash
// [0x5][0 0 0 0 0 0 0 1][fd1af8cec6c67fcf156f1b61fdf91ebc04d05484d007436e75342fc05bbff35a]
// An attestation is an event multiple people are voting on, this function needs the claim
// details because each Attestation is aggregating all claims of a specific event, lets say
// validator X and validator y where making different claims about the same event nonce
// Note that the claim hash does NOT include the claimer address and only identifies an event
func GetAttestationKeyWithHash(eventNonce uint64, claimHash []byte) []byte {
	// key := make([]byte, len(OracleAttestationKey)+len(UInt64Bytes(0))+len(claimHash))
	// copy(key[0:], OracleAttestationKey)
	// copy(key[len(OracleAttestationKey):], UInt64Bytes(eventNonce))
	// copy(key[len(OracleAttestationKey)+len(UInt64Bytes(0)):], claimHash)
	// return key
	return append(sdk.Uint64ToBigEndian(eventNonce), claimHash...)
}

// GetOutgoingTxPoolKey returns the following key format
// prefix  hyperionId        id
// [0x6][0 0 0 0 0 0 0 1][0 0 0 0 0 0 0 1]
func GetOutgoingTxPoolKey(hyperionId uint64, id uint64) []byte {
	buf := make([]byte, 0, len(OutgoingTXPoolKey)+8+8)
	buf = append(buf, OutgoingTXPoolKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, sdk.Uint64ToBigEndian(id)...)

	return buf
}

// GetOutgoingTxBatchKey returns the following key format
// prefix      hyperionId                nonce                     eth-contract-address
// [0xa][0 0 0 0 0 0 0 1][0 0 0 0 0 0 0 1][0xc783df8a850f42e7F7e57013759C285caa701eB6]
func GetOutgoingTxBatchKey(tokenContract common.Address, nonce uint64, hyperionId uint64) []byte {
	buf := make([]byte, 0, len(OutgoingTXBatchKey)+8+8+ETHContractAddressLen)
	buf = append(buf, OutgoingTXBatchKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, UInt64Bytes(nonce)...)
	buf = append(buf, tokenContract.Bytes()...)

	return buf
}

// GetOutgoingTxBatchBlockKey returns the following key format
// prefix  hyperionId   blockheight
// [0xb][0 0 0 0 0 0 0 1][0 0 0 0 2 1 4 3]
func GetOutgoingTxBatchBlockKey(hyperionId uint64, block uint64) []byte {
	buf := make([]byte, 0, len(OutgoingTXBatchBlockKey)+8+8)
	buf = append(buf, OutgoingTXBatchBlockKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, UInt64Bytes(block)...)
	return buf
}

// GetBatchConfirmKey returns the following key format
// prefix    hyperionId       eth-contract-address                BatchNonce                       Validator-address
// [0xe1][0 0 0 0 0 0 0 1][0xc783df8a850f42e7F7e57013759C285caa701eB6][0 0 0 0 0 0 0 1][cosmosvaloper1ahx7f8wyertuus9r20284ej0asrs085case3kn]
// TODO this should be a sdk.ValAddress
func GetBatchConfirmKey(hyperionId uint64, tokenContract common.Address, batchNonce uint64, validator sdk.AccAddress) []byte {
	buf := make([]byte, 0, len(BatchConfirmKey)+8+ETHContractAddressLen+8+len(validator))
	buf = append(buf, BatchConfirmKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, tokenContract.Bytes()...)
	buf = append(buf, UInt64Bytes(batchNonce)...)
	buf = append(buf, validator.Bytes()...)

	return buf
}

// GetFeeSecondIndexKey returns the following key format
// prefix    hyperionId        eth-contract-address            					fee_amount
// [0x9][0 0 0 0 0 0 0 1][0xc783df8a850f42e7F7e57013759C285caa701eB6][0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0]
func GetFeeSecondIndexKey(hyperionId uint64, tokenContract common.Address, fee *ERC20Token) []byte {
	buf := make([]byte, 0, len(SecondIndexOutgoingTXFeeKey)+8+ETHContractAddressLen+32)
	buf = append(buf, SecondIndexOutgoingTXFeeKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, tokenContract.Bytes()...)

	// sdk.BigInt represented as a zero-extended big-endian byte slice (32 bytes)
	amount := make([]byte, 32)
	amount = fee.Amount.BigInt().FillBytes(amount)
	buf = append(buf, amount...)

	return buf
}

func GetPrefixRangeForGetFeeSecondIndexKeyOnSpecificalTokenContract(hyperionId uint64, tokenContract common.Address) []byte {
	return append(UInt64Bytes(hyperionId), tokenContract.Bytes()...)
}

func GetPrefixRangeForGetFeeSecondIndexKey(hyperionId uint64) []byte {
	return UInt64Bytes(hyperionId)
}

// GetLastEventByValidatorKey indexes lateset event by validator
// GetLastEventByValidatorKey returns the following key format
// prefix    hyperionId        cosmos-validator
// [0x0][0,0,0,0,0,0,0,1][cosmos1ahx7f8wyertuus9r20284ej0asrs085case3kn]
func GetLastEventByValidatorKey(hyperionId uint64, validator sdk.ValAddress) []byte {
	buf := make([]byte, 0, len(LastEventByValidatorKey)+8+len(validator))
	buf = append(buf, LastEventByValidatorKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, validator.Bytes()...)

	return buf
}

func GetCosmosDenomToTokenAddressKey(hyperionId uint64, denom string) []byte {
	buf := make([]byte, 0, len(DenomToTokenAddressKey)+8+len(denom))
	buf = append(buf, DenomToTokenAddressKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, denom...)

	return buf
}

func GetTokenAddressToCosmosDenomKey(hyperionId uint64, tokenContract common.Address) []byte {
	buf := make([]byte, 0, len(TokenAddressToDenomKey)+8+ETHContractAddressLen)
	buf = append(buf, TokenAddressToDenomKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, tokenContract.Bytes()...)

	return buf
}

// GetPastEthSignatureCheckpointKey returns the following key format
// prefix    checkpoint
// [0x0][ checkpoint bytes ]
func GetPastEthSignatureCheckpointKey(hyperionId uint64, checkpoint common.Hash) []byte {
	buf := make([]byte, 0, len(PastEthSignatureCheckpointKey)+8+len(checkpoint))
	buf = append(buf, PastEthSignatureCheckpointKey...)
	buf = append(buf, UInt64Bytes(hyperionId)...)
	buf = append(buf, checkpoint[:]...)
	return buf
}

func GetLastOutgoingBatchIDKey(hyperionId uint64) []byte {
	return append(KeyLastOutgoingBatchID, UInt64Bytes(hyperionId)...)
}

func GetLastTXPoolIDKey(hyperionId uint64) []byte {
	return append(KeyLastTXPoolID, UInt64Bytes(hyperionId)...)
}

func GetFinalizedTxKey(ethereumAddress common.Address, txIndex uint64) []byte {
	buf := make([]byte, 0, ETHContractAddressLen+8)
	buf = append(buf, ethereumAddress.Bytes()...)
	buf = append(buf, UInt64Bytes(txIndex)...)
	return buf
}

func GetFinalizedTxAddressPrefixKey(ethereumAddress common.Address) []byte {
	buf := make([]byte, 0, ETHContractAddressLen)
	buf = append(buf, ethereumAddress.Bytes()...)
	return buf
}

func GetFinalizedTxAddressAndTxIndexPrefixKey(ethereumAddress common.Address, txIndex uint64) []byte {
	buf := make([]byte, 0, ETHContractAddressLen+8)
	buf = append(buf, ethereumAddress.Bytes()...)
	buf = append(buf, UInt64Bytes(txIndex)...)
	return buf
}
