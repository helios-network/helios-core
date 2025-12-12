package types

import (
	"math/big"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

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
	Address       common.Address `json:"address"`
	Denom         string         `json:"denom"`
	Symbol        string         `json:"symbol"`
	Balance       *hexutil.Big   `json:"balance"`
	BalanceUI     string         `json:"balanceUI"`
	Decimals      uint32         `json:"decimals"`
	Description   string         `json:"description"`
	OriginChainId string         `json:"originChainId"`
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
	TotalBoost       string              `json:"totalBoost"`
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
	BoostPercentage         string                   `json:"boostPercentage"`
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

// ValidatorAssetRPC represents an asset with contract address
type ValidatorAssetRPC struct {
	Denom           string                `json:"denom"`
	BaseAmount      cosmossdk_io_math.Int `json:"baseAmount"`
	WeightedAmount  cosmossdk_io_math.Int `json:"weightedAmount"`
	ContractAddress string                `json:"contractAddress"`
}

type ValidatorWithCommissionAndAssetsRPC struct {
	Validator  ValidatorRPC           `json:"validator"`
	Assets     []ValidatorAssetRPC    `json:"assets"`
	Commission ValidatorCommissionRPC `json:"commission"`
}

type ValidatorWithAssetsAndCommissionAndDelegationRPC struct {
	Validator  ValidatorRPC           `json:"validator"`
	Assets     []ValidatorAssetRPC    `json:"assets"`
	Commission ValidatorCommissionRPC `json:"commission"`
	Delegation DelegationRPC          `json:"delegation"`
}

type ValidatorsWithAssetsAndCommissionAndDelegationRPC struct {
	Validators []ValidatorWithAssetsAndCommissionAndDelegationRPC `json:"validators"`
}

type ValidatorWithAssetsRPC struct {
	Validator ValidatorRPC        `json:"validator"`
	Assets    []ValidatorAssetRPC `json:"assets"`
}

type WhitelistedAssetRPC struct {
	Denom                         string                `json:"denom"`
	BaseWeight                    uint64                `json:"baseWeight"`
	ChainId                       string                `json:"chainId"`
	ChainName                     string                `json:"chainName"`
	Decimals                      uint64                `json:"decimals"`
	Symbol                        string                `json:"symbol"`
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
	Paused                  bool   `json:"paused"`
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

type ProposalVoteOptionRPC struct {
	Option string `json:"option"`
	Weight string `json:"weight"`
}

type ProposalVoteRPC struct {
	Voter    string                  `json:"voter"`
	Options  []ProposalVoteOptionRPC `json:"options"`
	Metadata string                  `json:"metadata"`
}

type ProposalRPC struct {
	Id                 uint64                   `json:"id"`
	StatusCode         string                   `json:"statusCode"`
	Status             string                   `json:"status"`
	Proposer           string                   `json:"proposer"`
	Title              string                   `json:"title"`
	Metadata           string                   `json:"metadata"`
	Summary            string                   `json:"summary"`
	Details            []map[string]interface{} `json:"details"`
	Options            []ProposalVoteOptionRPC  `json:"options"`
	VotingStartTime    time.Time                `json:"votingStartTime"`
	VotingEndTime      time.Time                `json:"votingEndTime"`
	SubmitTime         time.Time                `json:"submitTime"`
	TotalDeposit       sdk.Coins                `json:"totalDeposit"`
	MinDeposit         sdk.Coins                `json:"minDeposit"`
	FinalTallyResult   govtypes.TallyResult     `json:"finalTallyResult"`
	CurrentTallyResult govtypes.TallyResult     `json:"currentTallyResult"`
}

type ValidatorAPYDetailsRPC struct {
	ValidatorAddress           string `json:"validatorAddress"`
	DelegatorAPY               string `json:"delegatorAPY"`
	ParticipationProbability   string `json:"participationProbability"`
	WeightedShares             string `json:"weightedShares"`
	TotalNetworkWeightedShares string `json:"totalNetworkWeightedShares"`
	SharePercentage            string `json:"sharePercentage"`
	RewardsPerBlock            string `json:"rewardsPerBlock"`
	RewardsPerEpoch            string `json:"rewardsPerEpoch"`
	RewardsPerDay              string `json:"rewardsPerDay"`
	AnnualRewards              string `json:"annualRewards"`
	CommissionRate             string `json:"commissionRate"`
	CommunityTax               string `json:"communityTax"`
}

type MsgCatalogEntry struct {
	TypeURL       string                 `json:"type_url"`
	ProtoFullName string                 `json:"proto_full_name"`
	Module        string                 `json:"module"`
	Service       string                 `json:"service"`
	Method        string                 `json:"method"`
	RequiresAuth  bool                   `json:"requires_authority"`
	JSONTemplate  map[string]interface{} `json:"json_template"`
}
