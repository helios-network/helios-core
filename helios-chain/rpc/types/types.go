package types

import (
	"math/big"
	"time"

	cosmossdk_io_math "cosmossdk.io/math"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// Copied the Account and StorageResult types since they are registered under an
// internal pkg on geth.

// TokenBalance représente la balance d'un token spécifique
type TokenBalance struct {
	Address     common.Address `json:"address"`
	Denom       string         `json:"denom"`
	Symbol      string         `json:"symbol"`
	Balance     *hexutil.Big   `json:"balance"`
	BalanceUI   string         `json:"balanceUI"`
	Decimals    uint32         `json:"decimals"`
	Description string         `json:"description"`
}

type TokenDetails struct {
	Address       common.Address `json:"address"`
	Denom         string         `json:"denom"`
	Symbol        string         `json:"symbol"`
	TotalSupply   *hexutil.Big   `json:"totalSupply"`
	TotalSupplyUI string         `json:"totalSupplyUI"`
	Decimals      uint32         `json:"decimals"`
	Description   string         `json:"description"`
	Logo          string         `json:"logo"`
	Holders       uint64         `json:"holders"`
}

type ChainSize struct {
	Bytes     int64 `json:"bytes"`
	MegaBytes int64 `json:"megaBytes"`
	GigaBytes int64 `json:"gigaBytes"`
	Terabytes int64 `json:"terabytes"`
}

// AccountResult struct for account proof
type AccountResult struct {
	Address      common.Address  `json:"address"`
	AccountProof []string        `json:"accountProof"`
	Balance      *hexutil.Big    `json:"balance"`
	CodeHash     common.Hash     `json:"codeHash"`
	Nonce        hexutil.Uint64  `json:"nonce"`
	StorageHash  common.Hash     `json:"storageHash"`
	StorageProof []StorageResult `json:"storageProof"`
}

// StorageResult defines the format for storage proof return
type StorageResult struct {
	Key   string       `json:"key"`
	Value *hexutil.Big `json:"value"`
	Proof []string     `json:"proof"`
}

// RPCTransaction represents a transaction that will serialize to the RPC representation of a transaction
type RPCTransaction struct {
	BlockHash        *common.Hash         `json:"blockHash"`
	BlockNumber      *hexutil.Big         `json:"blockNumber"`
	From             common.Address       `json:"from"`
	Gas              hexutil.Uint64       `json:"gas"`
	GasPrice         *hexutil.Big         `json:"gasPrice"`
	GasFeeCap        *hexutil.Big         `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big         `json:"maxPriorityFeePerGas,omitempty"`
	Hash             common.Hash          `json:"hash"`
	Input            hexutil.Bytes        `json:"input"`
	Nonce            hexutil.Uint64       `json:"nonce"`
	To               *common.Address      `json:"to"`
	TransactionIndex *hexutil.Uint64      `json:"transactionIndex"`
	Value            *hexutil.Big         `json:"value"`
	Type             hexutil.Uint64       `json:"type"`
	Accesses         *ethtypes.AccessList `json:"accessList,omitempty"`
	ChainID          *hexutil.Big         `json:"chainId,omitempty"`
	V                *hexutil.Big         `json:"v"`
	R                *hexutil.Big         `json:"r"`
	S                *hexutil.Big         `json:"s"`
}

type ParsedRPCTransaction struct {
	RawTransaction RPCTransaction         `json:rawTransaction`
	ParsedInfo     map[string]interface{} `json:parsedInfo`
}

type AccountTokensBalance struct {
	TotalCount uint64         `json:totalCount`
	Balances   []TokenBalance `json:balances`
}

// StateOverride is the collection of overridden accounts.
type StateOverride map[common.Address]OverrideAccount

// OverrideAccount indicates the overriding fields of account during the execution of
// a message call.
// Note, state and stateDiff can't be specified at the same time. If state is
// set, message execution will only use the data in the given state. Otherwise
// if statDiff is set, all diff will be applied first and then execute the call
// message.
type OverrideAccount struct {
	Nonce     *hexutil.Uint64              `json:"nonce"`
	Code      *hexutil.Bytes               `json:"code"`
	Balance   **hexutil.Big                `json:"balance"`
	State     *map[common.Hash]common.Hash `json:"state"`
	StateDiff *map[common.Hash]common.Hash `json:"stateDiff"`
}

type FeeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
	BaseFee      []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
}

// SignTransactionResult represents a RLP encoded signed transaction.
type SignTransactionResult struct {
	Raw hexutil.Bytes         `json:"raw"`
	Tx  *ethtypes.Transaction `json:"tx"`
}

type OneFeeHistory struct {
	BaseFee, NextBaseFee *big.Int   // base fee for each block
	Reward               []*big.Int // each element of the array will have the tip provided to miners for the percentile given
	GasUsedRatio         float64    // the ratio of gas used to the gas limit for each block
}

type DelegationAsset struct {
	Denom           string                `json:"denom"`
	BaseAmount      cosmossdk_io_math.Int `json:"baseAmount"`
	Amount          cosmossdk_io_math.Int `json:"amount"`
	WeightedAmount  cosmossdk_io_math.Int `json:"weightedAmount"`
	ContractAddress string                `json:"contractAddress"`
}

type DelegationRewardRPC struct {
	Denom           string                `json:"denom"`
	Amount          cosmossdk_io_math.Int `json:"amount"`
	ContractAddress string                `json:"contractAddress"`
}

type ValidatorCommissionRPC struct {
	Denom           string                `json:"denom"`
	Amount          cosmossdk_io_math.Int `json:"amount"`
	ContractAddress string                `json:"contractAddress"`
}

type ValidatorRewardRPC struct {
	Denom  string                `json:"denom"`
	Amount cosmossdk_io_math.Int `json:"amount"`
}

type DelegationRPC struct {
	ValidatorAddress string              `json:"validatorAddress"`
	Shares           string              `json:"shares"`
	Assets           []DelegationAsset   `json:"assets"`
	Rewards          DelegationRewardRPC `json:"rewards"`
}

type ValidatorRPC struct {
	ValidatorAddress        string                   `json:"validatorAddress"`
	Shares                  string                   `json:"shares"`
	Moniker                 string                   `json:"moniker"`
	Commission              stakingtypes.Commission  `json:"commission"`
	Description             stakingtypes.Description `json:"description"`
	Status                  stakingtypes.BondStatus  `json:"status"`
	UnbondingHeight         int64                    `json:"unbondingHeight"`
	UnbondingIds            []uint64                 `json:"unbondingIds"`
	Jailed                  bool                     `json:"jailed"`
	UnbondingOnHoldRefCount int64                    `json:"unbondingOnHoldRefCount"`
	UnbondingTime           time.Time                `json:"unbondingTime"`
	MinSelfDelegation       cosmossdk_io_math.Int    `json:"minSelfDelegation"`
	Apr                     string                   `json:"apr"`
	MinDelegation           cosmossdk_io_math.Int    `json:"minDelegation"`
	DelegationAuthorization bool                     `json:"delegationAuthorization"`
	TotalBoost              string                   `json:"totalBoost"`
}

type ValidatorWithDelegationRPC struct {
	Validator  ValidatorRPC  `json:"validator"`
	Delegation DelegationRPC `json:"delegation"`
}

type ValidatorWithCommissionRPC struct {
	Validator  ValidatorRPC           `json:"validator"`
	Commission ValidatorCommissionRPC `json:"commission"`
}

type ValidatorWithCommissionAndDelegationRPC struct {
	Validator  ValidatorRPC           `json:"validator"`
	Delegation DelegationRPC          `json:"delegation"`
	Commission ValidatorCommissionRPC `json:"commission"`
}

type WhitelistedAssetRPC struct {
	Denom                         string                `json:"denom"`
	BaseWeight                    uint64                `json:"baseWeight"`
	ChainId                       string                `json:"chainId"`
	Decimals                      uint64                `json:"decimals"`
	Metadata                      string                `json:"metadata"`
	ContractAddress               string                `json:"contractAddress"`
	TotalShares                   cosmossdk_io_math.Int `json:"totalShares"`
	NetworkPercentageSecurisation string                `json:"networkPercentageSecurisation"`
}

type HyperionChainRPC struct {
	HyperionContractAddress string `json:"hyperionContractAddress"`
	ChainId                 uint64 `json:"chainId"`
	Name                    string `json:"name"`
	ChainType               string `json:"chainType"`
	Logo                    string `json:"logo"`
	HyperionId              uint64 `json:"hyperionId"`
}

// ValidatorSignature struct for block signature information
type ValidatorSignature struct {
	Address          string         `json:"address"`
	Signed           bool           `json:"signed"`
	IndexOffset      int64          `json:"indexOffset"`
	TotalTokens      string         `json:"totalTokens"`
	AssetWeights     []*AssetWeight `json:"assetWeights"`
	EpochNumber      int64          `json:"epochNumber"`
	Status           string         `json:"status"`
	Jailed           bool           `json:"jailed"`
	MissedBlockCount int64          `json:"missedBlockCount"`
}

// AssetWeight struct for asset weight information
type AssetWeight struct {
	Denom          string `json:"denom"`
	BaseAmount     string `json:"baseAmount"`
	WeightedAmount string `json:"weightedAmount"`
}

// EpochValidatorSignature represents a validator's signature at a specific height
type EpochValidatorSignature struct {
	Signature bool  `json:"signature"`
	Height    int64 `json:"height"`
}

// EpochValidatorDetail contains complete information about a validator in an epoch
type EpochValidatorDetail struct {
	ValidatorAddress       string                     `json:"validatorAddress"`
	OperatorAddress        string                     `json:"operatorAddress"`
	BlocksSigned           []*EpochValidatorSignature `json:"blocksSigned"`
	BlocksMissed           []int64                    `json:"blocksMissed"`
	BondedTokens           string                     `json:"bondedTokens"`
	Status                 string                     `json:"status"`
	IsSlashed              bool                       `json:"isSlashed"`
	IsJailed               bool                       `json:"isJailed"`
	AssetWeights           []*AssetWeight             `json:"assetWeights"`
	MissedBlockCount       int64                      `json:"missedBlockCount"`
	SigningInfoStartHeight string                     `json:"signingInfoStartHeight"`
	CommissionRate         string                     `json:"commissionRate"`
	VotingPower            string                     `json:"votingPower"`
}

// EpochCompleteResponse contains complete information about an epoch
type EpochCompleteResponse struct {
	Epoch                uint64                  `json:"epoch"`
	EpochLength          uint64                  `json:"epochLength"`
	StartHeight          int64                   `json:"startHeight"`
	EndHeight            int64                   `json:"endHeight"`
	CurrentHeight        int64                   `json:"currentHeight"`
	BlocksValidated      int64                   `json:"blocksValidated"`
	BlocksRemaining      int64                   `json:"blocksRemaining"`
	BlocksUntilNextEpoch int64                   `json:"blocksUntilNextEpoch"`
	Validators           []*EpochValidatorDetail `json:"validators"`
	TotalTokens          string                  `json:"totalTokens"`
	TotalVotingPower     string                  `json:"totalVotingPower"`
}
