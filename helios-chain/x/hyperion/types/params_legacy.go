package types

import paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

var (
	// ParamsCounterpartyChainParams stores the couterparty chain params
	ParamsCounterpartyChainParams = []byte("CounterpartyChainParams")

	// ParamsStoreKeyHyperionID stores the hyperion id
	ParamsStoreKeyHyperionID = []byte("HyperionID")

	// ParamsStoreKeyContractHash stores the contract hash
	ParamsStoreKeyContractHash = []byte("ContractHash")

	// ParamsStoreKeyBridgeContractAddress stores the contract address
	ParamsStoreKeyBridgeContractAddress = []byte("BridgeContractAddress")

	// ParamsStoreKeyBridgeContractStartHeight stores the bridge contract deployed height
	ParamsStoreKeyBridgeContractStartHeight = []byte("BridgeContractChainHeight")

	// ParamsStoreKeyBridgeContractChainID stores the bridge chain id
	ParamsStoreKeyBridgeContractChainID = []byte("BridgeChainID")

	// ParamsStoreKeyCosmosCoinDenom stores native cosmos coin denom
	ParamsStoreKeyCosmosCoinDenom = []byte("CosmosCoinDenom")

	// ParamsStoreKeyCosmosCoinErc20Contract store L1 erc20 contract address of cosmos native coin
	ParamsStoreKeyCosmosCoinErc20Contract = []byte("CosmosCoinErc20Contract")

	// ParamsStoreKeySignedValsetsWindow stores the signed blocks window
	ParamsStoreKeySignedValsetsWindow = []byte("SignedValsetsWindow")

	// ParamsStoreKeySignedBatchesWindow stores the signed blocks window
	ParamsStoreKeySignedBatchesWindow = []byte("SignedBatchesWindow")

	// ParamsStoreKeySignedClaimsWindow stores the signed blocks window
	ParamsStoreKeySignedClaimsWindow = []byte("SignedClaimsWindow")

	// ParamsStoreKeyTargetBatchTimeout stores
	ParamsStoreKeyTargetBatchTimeout = []byte("TargetBatchTimeout")

	// ParamsStoreKeyAverageBlockTime stores the average block time of the Helios Chain in milliseconds
	ParamsStoreKeyAverageBlockTime = []byte("AverageBlockTime")

	// ParamsStoreKeyAverageEthereumBlockTime stores the average block time of Ethereum in milliseconds
	ParamsStoreKeyAverageEthereumBlockTime = []byte("AverageEthereumBlockTime")

	// ParamsStoreSlashFractionValset stores the slash fraction valset
	ParamsStoreSlashFractionValset = []byte("SlashFractionValset")

	// ParamsStoreSlashFractionBatch stores the slash fraction Batch
	ParamsStoreSlashFractionBatch = []byte("SlashFractionBatch")

	// ParamsStoreSlashFractionClaim stores the slash fraction Claim
	ParamsStoreSlashFractionClaim = []byte("SlashFractionClaim")

	// ParamsStoreSlashFractionConflictingClaim stores the slash fraction ConflictingClaim
	ParamsStoreSlashFractionConflictingClaim = []byte("SlashFractionConflictingClaim")

	// ParamStoreUnbondSlashingValsetsWindow stores unbond slashing valset window
	ParamStoreUnbondSlashingValsetsWindow = []byte("UnbondSlashingValsetsWindow")

	// ParamStoreSlashFractionBadEthSignature stores the amount by which a validator making a fraudulent eth signature will be slashed
	ParamStoreSlashFractionBadEthSignature = []byte("SlashFractionBadEthSignature")

	// ParamStoreValsetRewardAmount is the amount of the coin, both denom and amount to issue
	// to a relayer when they relay a valset
	ParamStoreValsetRewardAmount = []byte("ValsetReward")

	ParamStoreAdmins = []byte("Admins")

	// Ensure that params implements the proper interface
	_ paramtypes.ParamSet = &Params{}
)

// ParamKeyTable for auth module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of auth module's parameters.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamsCounterpartyChainParams, &p.CounterpartyChainParams, validateCounterpartyChainParams),
	}
}
