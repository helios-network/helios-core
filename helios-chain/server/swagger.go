package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/swaggest/swgui"
	"github.com/swaggest/swgui/v3cdn"

	"github.com/gorilla/mux"

	cosmossdk_io_math "cosmossdk.io/math"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	rpctypes "helios-core/helios-chain/rpc/types"
	chronostypes "helios-core/helios-chain/x/chronos/types"
	evmtypes "helios-core/helios-chain/x/evm/types"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"

	svrconfig "helios-core/helios-chain/server/config"

	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func getDefaultForType(methodName string, kind reflect.Kind, t reflect.Value) (string, interface{}) {
	switch t.String() {
	case "<*common.Address Value>":
		return "string", "0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd"
	case "<common.Address Value>":
		return "string", "0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd"
	case "<*common.Hash Value>":
		return "string", "0x6622ef5bfeeefaae96ac92addce33088c06aee9838f4add1e3e0a6795b7c2a97"
	case "<*types.BlockNumberOrHash Value>":
		return "string", "latest"
	case "<*types.AccountResult Value>":
		return "types.AccountResult", rpctypes.AccountResult{}
	case "<*types.TransactionArgs Value>":
		addr := common.HexToAddress("0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd")
		nonce := hexutil.Uint64(0)
		gasPrice := hexutil.Big(*big.NewInt(2000000000))
		value := hexutil.Big(*big.NewInt(0))
		gas := hexutil.Uint64(21000)
		data := hexutil.Bytes([]byte{})

		chainId := hexutil.Big(*big.NewInt(int64(evmtypes.GetChainConfig().ChainId)))
		return "evmtypes.TransactionArgs", evmtypes.TransactionArgs{
			From:       &addr,
			To:         &addr,
			Gas:        &gas,
			Nonce:      &nonce,
			GasPrice:   &gasPrice,
			Value:      &value,
			Data:       &data,
			Input:      nil,
			AccessList: nil,
			ChainID:    &chainId,
		}
	case "<*types.RPCTransaction Value>":
		return "rpctypes.RPCTransaction", rpctypes.RPCTransaction{}
	case "<*types.StateOverride Value>":
		return "rpctypes.StateOverride", rpctypes.StateOverride{}
	case "<*types.CronTransactionRPC Value>":
		return "chronostypes.CronTransactionRPC", chronostypes.CronTransactionRPC{}
	case "<*types.CronTransactionReceiptRPC Value>":
		return "chronostypes.CronTransactionReceiptRPC", chronostypes.CronTransactionReceiptRPC{}
	case "<*hexutil.Bytes Value>":
		return "string", "0x0"
	case "<*rpc.DecimalOrHex Value>":
		return "string", "0x1"
	case "<*rpc.BlockNumber Value>":
		return "string", "latest"
	case "<*types.BlockNumber Value>":
		return "string", "latest"
	case "<*hexutil.Uint64 Value>":
		return "string", "0x0"
	case "<*uint64 Value>":
		return "number", 0
	case "<*int Value>":
		return "number", 0
	case "<*hexutil.Big Value>":
		return "string", "0x0"
	case "<*apitypes.TypedData Value>":
		return "apitypes.TypedData", map[string]interface{}{}
	case "<*types.Log Value>":
		return "evmtypes.Log", evmtypes.Log{}
	case "<*string Value>":
		return "string", ""
	case "<*hexutil.Uint Value>":
		return "string", "0x0"
	case "<uint8 Value>":
		return "number", 0
	case "<*types.ChainSize Value>":
		return "rpctypes.ChainSize", rpctypes.ChainSize{}
	case "<*[]float64 Value>":
		return "array", []float64{0}
	case "<*types.FeeHistoryResult Value>":
		return "rpctypes.FeeHistoryResult", rpctypes.FeeHistoryResult{}
	case "<*types.SignTransactionResult Value>":
		return "rpctypes.SignTransactionResult", rpctypes.SignTransactionResult{}
	case "<*types.Cron Value>":
		return "chronostypes.Cron", chronostypes.Cron{}
	case "<types.Cron Value>":
		return "chronostypes.Cron", chronostypes.Cron{}
	case "<types.TokenBalance Value>":
		return "rpctypes.TokenBalance", rpctypes.TokenBalance{}
	case "<*[]interface {} Value>":
		return "[]interface", []interface{}{}
	case "<map[string]interface {} Value>":
		return "map[string]interface", map[string]interface{}{}
	case "<*map[string]interface {} Value>":
		return "map[string]interface", map[string]interface{}{}
	case "<*interface {} Value>":
		return "interface", []interface{}{}
	case "<*context.Context Value>":
		return "context.Context", []interface{}{}
	case "<*bool Value>":
		return "bool", false
	case "<*[]string Value>":
		return "[]string", []string{""}
	case "<types.WhitelistedAssetRPC Value>":
		return "rpctypes.WhitelistedAssetRPC", rpctypes.WhitelistedAssetRPC{
			Denom:                         "default_denom",
			BaseWeight:                    0,
			ChainId:                       "default_chain_id",
			ChainName:                     "default_chain_name",
			Decimals:                      0,
			Symbol:                        "default_symbol",
			ContractAddress:               "0x0000000000000000000000000000000000000000",
			TotalShares:                   cosmossdk_io_math.NewInt(0),
			NetworkPercentageSecurisation: "0%",
		}
	case "<types.DelegationRPC Value>":
		return "rpctypes.DelegationRPC", rpctypes.DelegationRPC{
			ValidatorAddress: "0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd",
			Shares:           "0",
			Assets: []rpctypes.DelegationAsset{
				{
					Denom:          "asset_denom",
					BaseAmount:     cosmossdk_io_math.NewInt(0),
					Amount:         cosmossdk_io_math.NewInt(0),
					WeightedAmount: cosmossdk_io_math.NewInt(0),
				},
			},
			Rewards: rpctypes.DelegationRewardRPC{
				Denom:  "default_reward_denom",
				Amount: cosmossdk_io_math.NewInt(0),
			},
		}
	case "<*types.DelegationRPC Value>":
		return "rpctypes.DelegationRPC", rpctypes.DelegationRPC{
			ValidatorAddress: "0x9bFE7f4Aae74EF013e821ef93c092c2d42eac4dd",
			Shares:           "0",
			Assets: []rpctypes.DelegationAsset{
				{
					Denom:          "asset_denom",
					BaseAmount:     cosmossdk_io_math.NewInt(0),
					Amount:         cosmossdk_io_math.NewInt(0),
					WeightedAmount: cosmossdk_io_math.NewInt(0),
				},
			},
			Rewards: rpctypes.DelegationRewardRPC{
				Denom:  "default_reward_denom",
				Amount: cosmossdk_io_math.NewInt(0),
			},
		}
	case "<types.ValidatorRPC Value>":
		return "rpctypes.ValidatorRPC", rpctypes.ValidatorRPC{
			ValidatorAddress: "0x0000000000000000000000000000000000000000",
			Shares:           "0",
			Moniker:          "default_validator",
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          cosmossdk_io_math.LegacyNewDec(0), // Taux de commission par défaut
					MaxRate:       cosmossdk_io_math.LegacyNewDec(0), // Taux maximum par défaut
					MaxChangeRate: cosmossdk_io_math.LegacyNewDec(0), // Taux de changement maximum par défaut
				},
			},
			Description: stakingtypes.Description{
				Moniker:         "default_validator",
				Identity:        "",
				Website:         "",
				SecurityContact: "",
				Details:         "",
			},
			Status:                  stakingtypes.Bonded, // Statut par défaut
			UnbondingHeight:         0,
			UnbondingIds:            []uint64{},
			Jailed:                  false,
			UnbondingOnHoldRefCount: 0,
			UnbondingTime:           time.Time{},
			MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
			Apr:                     "0%", // APR par défaut
		}
	case "<*types.ValidatorRPC Value>":
		return "rpctypes.ValidatorRPC", rpctypes.ValidatorRPC{
			ValidatorAddress: "0x0000000000000000000000000000000000000000",
			Shares:           "0",
			Moniker:          "default_validator",
			Commission: stakingtypes.Commission{
				CommissionRates: stakingtypes.CommissionRates{
					Rate:          cosmossdk_io_math.LegacyNewDec(0), // Taux de commission par défaut
					MaxRate:       cosmossdk_io_math.LegacyNewDec(0), // Taux maximum par défaut
					MaxChangeRate: cosmossdk_io_math.LegacyNewDec(0), // Taux de changement maximum par défaut
				},
			},
			Description: stakingtypes.Description{
				Moniker:         "default_validator",
				Identity:        "",
				Website:         "",
				SecurityContact: "",
				Details:         "",
			},
			Status:                  stakingtypes.Bonded, // Statut par défaut
			UnbondingHeight:         0,
			UnbondingIds:            []uint64{},
			Jailed:                  false,
			UnbondingOnHoldRefCount: 0,
			UnbondingTime:           time.Time{},
			MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
			Apr:                     "0%", // APR par défaut
		}
	case "<*types.ValidatorCommissionRPC Value>":
		return "rpctypes.ValidatorCommissionRPC", rpctypes.ValidatorCommissionRPC{
			Denom:  "default_denom",             // Valeur par défaut pour le denom
			Amount: cosmossdk_io_math.NewInt(0), // Valeur par défaut pour le montant
		}
	case "<*types.ValidatorWithDelegationRPC Value>":
		return "rpctypes.ValidatorWithDelegationRPC", rpctypes.ValidatorWithDelegationRPC{
			Validator: rpctypes.ValidatorRPC{
				ValidatorAddress:        "0x0000000000000000000000000000000000000000",
				Shares:                  "0",
				Moniker:                 "default_validator",
				Commission:              stakingtypes.Commission{},
				Description:             stakingtypes.Description{},
				Status:                  stakingtypes.Bonded,
				UnbondingHeight:         0,
				UnbondingIds:            []uint64{},
				Jailed:                  false,
				UnbondingOnHoldRefCount: 0,
				UnbondingTime:           time.Time{},
				MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
				Apr:                     "0%",
			},
			Delegation: rpctypes.DelegationRPC{
				ValidatorAddress: "0x0000000000000000000000000000000000000000",
				Shares:           "0",
				Assets:           []rpctypes.DelegationAsset{},
				Rewards: rpctypes.DelegationRewardRPC{
					Denom:  "default_reward_denom",
					Amount: cosmossdk_io_math.NewInt(0),
				},
			},
		}
	case "<*types.ValidatorWithCommissionRPC Value>":
		return "rpctypes.ValidatorWithCommissionRPC", rpctypes.ValidatorWithCommissionRPC{
			Validator: rpctypes.ValidatorRPC{
				ValidatorAddress:        "0x0000000000000000000000000000000000000000",
				Shares:                  "0",
				Moniker:                 "default_validator",
				Commission:              stakingtypes.Commission{},
				Description:             stakingtypes.Description{},
				Status:                  stakingtypes.Bonded,
				UnbondingHeight:         0,
				UnbondingIds:            []uint64{},
				Jailed:                  false,
				UnbondingOnHoldRefCount: 0,
				UnbondingTime:           time.Time{},
				MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
				Apr:                     "0%",
			},
			Commission: rpctypes.ValidatorCommissionRPC{
				Denom:  "default_denom",
				Amount: cosmossdk_io_math.NewInt(0),
			},
		}
	case "<*types.ValidatorRewardRPC Value>":
		return "rpctypes.ValidatorRewardRPC", rpctypes.ValidatorRewardRPC{
			Amount: cosmossdk_io_math.NewInt(0),
			Denom:  "default_denom", // Valeur par défaut pour le denom
		}
	case "<*types.ValidatorWithCommissionAndDelegationRPC Value>":
		return "rpctypes.ValidatorWithCommissionAndDelegationRPC", rpctypes.ValidatorWithCommissionAndDelegationRPC{
			Validator: rpctypes.ValidatorRPC{
				ValidatorAddress:        "0x0000000000000000000000000000000000000000",
				Shares:                  "0",
				Moniker:                 "default_validator",
				Commission:              stakingtypes.Commission{},
				Description:             stakingtypes.Description{},
				Status:                  stakingtypes.Bonded,
				UnbondingHeight:         0,
				UnbondingIds:            []uint64{},
				Jailed:                  false,
				UnbondingOnHoldRefCount: 0,
				UnbondingTime:           time.Time{},
				MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
				Apr:                     "0%",
			},
			Delegation: rpctypes.DelegationRPC{
				ValidatorAddress: "0x0000000000000000000000000000000000000000",
				Shares:           "0",
				Assets:           []rpctypes.DelegationAsset{},
				Rewards: rpctypes.DelegationRewardRPC{
					Denom:  "default_reward_denom",
					Amount: cosmossdk_io_math.NewInt(0),
				},
			},
			Commission: rpctypes.ValidatorCommissionRPC{
				Denom:  "default_denom",
				Amount: cosmossdk_io_math.NewInt(0),
			},
		}
	case "<*types.ValidatorWithCommissionAndAssetsRPC Value>":
		return "rpctypes.ValidatorWithCommissionAndAssetsRPC", rpctypes.ValidatorWithCommissionAndAssetsRPC{
			Validator: rpctypes.ValidatorRPC{
				ValidatorAddress: "0x0000000000000000000000000000000000000000",
				Shares:           "0",
				Moniker:          "default_validator",
				Commission: stakingtypes.Commission{
					CommissionRates: stakingtypes.CommissionRates{
						Rate:          cosmossdk_io_math.LegacyNewDec(0),
						MaxRate:       cosmossdk_io_math.LegacyNewDec(0),
						MaxChangeRate: cosmossdk_io_math.LegacyNewDec(0),
					},
					UpdateTime: time.Now(),
				},
				Description: stakingtypes.Description{
					Moniker:         "default_validator",
					Identity:        "",
					Website:         "",
					SecurityContact: "",
					Details:         "",
				},
				Status:                  stakingtypes.Bonded,
				UnbondingHeight:         0,
				UnbondingIds:            []uint64{},
				Jailed:                  false,
				UnbondingOnHoldRefCount: 0,
				UnbondingTime:           time.Time{},
				MinSelfDelegation:       cosmossdk_io_math.NewInt(0),
				Apr:                     "0%",
			},
			Assets: []rpctypes.ValidatorAssetRPC{
				{
					Denom:           "default_denom",
					BaseAmount:      cosmossdk_io_math.NewInt(0),
					ContractAddress: "0x0000000000000000000000000000000000000000",
					WeightedAmount:  cosmossdk_io_math.NewInt(0),
				},
			},
		}
	case "<*types.OrchestratorData Value>":
		return "rpctypes.OrchestratorData", hyperiontypes.OrchestratorData{
			Orchestrator: "0x0000000000000000000000000000000000000000",
			OrchestratorHyperionData: []*hyperiontypes.OrchestratorHyperionData{
				{
					HyperionId:       0,
					MinimumTxFee:     cosmossdk_io_math.NewInt(0),
					MinimumBatchFee:  cosmossdk_io_math.NewInt(0),
					TotalSlashCount:  0,
					TotalSlashAmount: cosmossdk_io_math.NewInt(0),
					SlashData: []*hyperiontypes.SlashData{
						{
							SlashAmount:    cosmossdk_io_math.NewInt(0),
							SlashTimestamp: 0,
						},
					},
					TxOutTransfered:            0,
					TxInTransfered:             0,
					BatchCreated:               0,
					BatchConfirmed:             0,
					FeeCollected:               cosmossdk_io_math.NewInt(0),
					ExternalDataTxExecuted:     0,
					ExternalDataTxFeeCollected: cosmossdk_io_math.NewInt(0),
				},
			},
		}
	case "<*types.FullMetadata Value>":
		return "rpctypes.FullMetadata", banktypes.FullMetadata{
			Metadata: &banktypes.Metadata{
				Description: "default_description",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom: "default_denom",
					},
				},
				Base:    "default_base",
				Display: "default_display",
				Symbol:  "default_symbol",
				URI:     "default_uri",
				URIHash: "default_uri_hash",
			},
			HoldersCount: 0,
			TotalSupply:  cosmossdk_io_math.NewInt(0),
		}
	case "<[]types.FullMetadata Value>":
		return "[]rpctypes.FullMetadata", []banktypes.FullMetadata{
			{
				Metadata: &banktypes.Metadata{
					Description: "default_description",
				},
			},
		}
	case "<*types.AccountTokensBalance Value>":
		return "rpctypes.AccountTokensBalance", rpctypes.AccountTokensBalance{
			TotalCount: 0,
			Balances:   make([]rpctypes.TokenBalance, 0),
		}
	case "<*types.ValidatorSignature Value>":
		return "rpctypes.ValidatorSignature", rpctypes.ValidatorSignature{
			Address:          "0x0000000000000000000000000000000000000000",
			Signed:           false,
			IndexOffset:      0,
			TotalTokens:      "0",
			AssetWeights:     make([]*rpctypes.AssetWeight, 0),
			EpochNumber:      0,
			Status:           "default_status",
			Jailed:           false,
			MissedBlockCount: 0,
		}
	case "<*types.CronStatistics Value>":
		return "rpctypes.CronStatistics", chronostypes.CronStatistics{
			CronCount:              0,
			QueueCount:             0,
			ArchivedCrons:          0,
			RefundedLastBlockCount: 0,
			ExecutedLastBlockCount: 0,
		}
	case "<*types.EpochCompleteResponse Value>":
		return "rpctypes.EpochCompleteResponse", rpctypes.EpochCompleteResponse{
			Epoch:                0,
			EpochLength:          0,
			StartHeight:          0,
			EndHeight:            0,
			CurrentHeight:        0,
			BlocksValidated:      0,
			BlocksRemaining:      0,
			BlocksUntilNextEpoch: 0,
			Validators:           make([]*rpctypes.EpochValidatorDetail, 0),
			TotalTokens:          "0",
			TotalVotingPower:     "0",
		}
	case "<*types.TransferTx Value>":
		return "rpctypes.TransferTx", hyperiontypes.TransferTx{
			HyperionId:    0,
			Id:            0,
			Height:        0,
			Sender:        "0x0000000000000000000000000000000000000000",
			DestAddress:   "0x0000000000000000000000000000000000000000",
			ReceivedToken: &hyperiontypes.Token{},
			SentToken:     &hyperiontypes.Token{},
			ReceivedFee:   &hyperiontypes.Token{},
			SentFee:       &hyperiontypes.Token{},
			Status:        "default_status",
			Direction:     "default_direction",
			ChainId:       0,
			Proof:         &hyperiontypes.Proof{},
			TxHash:        "0x0000000000000000000000000000000000000000",
			Index:         0,
		}
	case "<*types.HyperionChainRPC Value>":
		return "rpctypes.HyperionChainRPC", rpctypes.HyperionChainRPC{
			HyperionContractAddress: "0x0000000000000000000000000000000000000000",
			ChainId:                 0,
			HyperionId:              0,
			Name:                    "default_chain_name",
			ChainType:               "default_chain_type",
			Logo:                    "default_chain_logo",
			Paused:                  false,
		}
	case "<*types.QueryHistoricalFeesResponse Value>":
		return "rpctypes.QueryHistoricalFeesResponse", hyperiontypes.QueryHistoricalFeesResponse{
			HistoricalFees: make([]*sdktypes.Coin, 0),
			Low:            &sdktypes.Coin{},
			High:           &sdktypes.Coin{},
			Average:        &sdktypes.Coin{},
		}
	case "<*types.ParsedRPCTransaction Value>":
		return "rpctypes.ParsedRPCTransaction", rpctypes.ParsedRPCTransaction{
			RawTransaction: rpctypes.RPCTransaction{},
			ParsedInfo:     make(map[string]interface{}),
		}
	case "<*types.ProposalVoteRPC Value>":
		return "rpctypes.ProposalVoteRPC", rpctypes.ProposalVoteRPC{
			Voter:    "0x0000000000000000000000000000000000000000",
			Options:  make([]rpctypes.ProposalVoteOptionRPC, 0),
			Metadata: "",
		}
	}
	fmt.Printf("[Swagger RPC] - No Mock method=%s kind=%s, type=%s", methodName, kind, t.String())
	return t.String(), t.Interface()
}

func generateInputOutput(structType interface{}) (map[string]interface{}, map[string]interface{}) {
	t := reflect.TypeOf(structType)
	paths := make(map[string]interface{})
	schemas := make(map[string]interface{})

	for i := 0; i < t.NumMethod(); i++ {
		method := t.Method(i)

		methodName := method.Name
		if len(methodName) > 0 {
			methodName = strings.ToLower(string(methodName[0])) + methodName[1:]
		}

		//////////////////////////////////////
		// INPUT's
		//////////////////////////////////////
		params := make([]map[string]interface{}, 0)

		for j := 1; j < method.Type.NumIn(); j++ {
			inType := method.Type.In(j)

			if inType.Kind() == reflect.Ptr {
				inType = inType.Elem()
			}

			t, d := getDefaultForType(methodName, inType.Kind(), reflect.New(inType))

			param := map[string]interface{}{
				"type":    t,
				"default": d,
			}
			params = append(params, param)
		}

		//////////////////////////////////////
		// OUTPUT's
		//////////////////////////////////////

		responseSchema := map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"default":    map[string]interface{}{},
		}
		// responseSchema := map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "default": map[string]interface{}{}}
		if method.Type.NumOut() > 0 {
			outType := method.Type.Out(0)
			if outType.Kind() == reflect.Ptr {
				outType = outType.Elem()
			}

			if outType.Kind() == reflect.Slice {
				sliceInstance := make([]interface{}, 0)
				elementInstance := reflect.New(outType.Elem()).Elem()

				t, d := getDefaultForType(methodName, elementInstance.Kind(), elementInstance)
				sliceInstance = append(sliceInstance, d)
				responseSchema["default"] = sliceInstance

				if _, exists := schemas[t]; !exists {
					properties := map[string]interface{}{}
					outType := reflect.TypeOf(d)

					if outType.Kind() == reflect.Struct {
						for i := 0; i < outType.NumField(); i++ {
							field := outType.Field(i)
							properties[field.Name] = map[string]interface{}{
								"type": field.Type.String(),
							}
						}
					}

					schemas[t] = map[string]interface{}{
						"type":       "object",
						"properties": properties,
						"example":    d,
					}
				}
				responseSchema = map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", t),
				}
			} else {
				outputInstance := reflect.New(outType)
				t, d := getDefaultForType(methodName, outputInstance.Kind(), outputInstance)
				responseSchema["default"] = d

				if _, exists := schemas[t]; !exists {
					properties := map[string]interface{}{}
					outType := reflect.TypeOf(d)

					if outType.Kind() == reflect.Struct {
						for i := 0; i < outType.NumField(); i++ {
							field := outType.Field(i)
							properties[field.Name] = map[string]interface{}{
								"type": field.Type.String(),
							}
						}
					}

					schemas[t] = map[string]interface{}{
						"type":       "object",
						"properties": properties,
						"example":    d,
					}
				}
				responseSchema = map[string]interface{}{
					"$ref": fmt.Sprintf("#/components/schemas/%s", t),
				}
			}
		}

		//////////////////////////////////////
		// Generate OpenAPI structure
		//////////////////////////////////////

		defaultValues := []interface{}{} // Créer un tableau pour stocker les valeurs par défaut

		// Supposons que params soit un tableau d'objets avec un champ "default"
		for _, param := range params {
			if param["default"] != nil { // Vérifiez si la valeur par défaut existe
				defaultValues = append(defaultValues, param["default"]) // Ajouter la valeur par défaut au tableau
			}
		}

		// Maintenant, vous pouvez assigner defaultValues à l'attribut "default"
		paramsFinal := map[string]interface{}{
			"type":    "array",
			"items":   params,
			"default": defaultValues, // Appliquer le tableau des valeurs par défaut
		}

		paths["/eth_"+methodName] = map[string]interface{}{
			"post": map[string]interface{}{
				"summary": fmt.Sprintf("eth_%s RPC method", methodName),
				"requestBody": map[string]interface{}{
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"jsonrpc": map[string]interface{}{"type": "string", "default": "2.0"},
									"method":  map[string]interface{}{"type": "string", "default": "eth_" + methodName},
									"params":  paramsFinal,
									"id":      map[string]interface{}{"type": "integer", "default": 1},
								},
								"required": []string{"jsonrpc", "method", "params", "id"},
							},
						},
					},
				},
				"responses": map[string]interface{}{
					"200": map[string]interface{}{"description": "Successful Response", "content": map[string]interface{}{"application/json": map[string]interface{}{"schema": responseSchema}}},
				},
			},
		}
		responseSchemaName := fmt.Sprintf("%sResponse", methodName)
		schemas[responseSchemaName] = responseSchema
	}

	return paths, schemas
}

func generateOpenAPI(paths map[string]interface{}, schemas map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Helios Core RPC",
			"version": "1.0.0",
			"description": `
			Helios Core RPC supports the following RPC protocols:

            * JSONRPC over HTTP
            * JSONRPC over WebSockets
			`,
		},
		"paths": paths,
		"components": map[string]interface{}{
			"schemas": schemas, // Ajout des schémas ici
		},
	}
}

func generateSwagger(ctx *server.Context, structType interface{}, router *mux.Router, srvConfig *svrconfig.Config) *v3cdn.Handler {
	rpcPath := "/"
	settingsUI := make(map[string]string)
	settingsUI["requestInterceptor"] = `function(request) {
		if (request.loadSpec) {
			return request;
		}
		var url = window.location.protocol + '//'+ window.location.host;
		var method = request.url.substring(url.length);
		var body = request.body ?? "[]";
		request.headers['Content-Type'] = 'application/json';
		request.url = url + '` + rpcPath + `';
		request.body = body; //'{"jsonrpc": "2.0", "method": "' + method + '", "id": 1, "params": ' + body + '}';
		return request;
	}`

	handler := v3cdn.NewHandlerWithConfig(swgui.Config{
		Title:       "Helios RPC",
		SwaggerJSON: "/docs/openapi.json",
		BasePath:    "/docs",
		SettingsUI:  settingsUI,
	})

	paths, schemas := generateInputOutput(structType)
	openAPI := generateOpenAPI(paths, schemas)
	jsonData, _ := json.MarshalIndent(openAPI, "", "  ")

	openApi := []byte(string(jsonData))
	router.HandleFunc("/docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(openApi)
	}).Methods("GET")
	router.HandleFunc("/docs", handler.ServeHTTP).Methods("GET")

	host, port, _ := net.SplitHostPort(parseURL(srvConfig.JSONRPC.Address))
	docsURL := fmt.Sprintf("http://%s:%s/docs", host, port)

	ctx.Logger.Info("SWAGGER API RPC Documentation", "url", docsURL)
	return handler
}

//go:embed grpc-openapi.json
var openapiFile embed.FS

func serveOpenAPI(w http.ResponseWriter, r *http.Request) {
	data, err := openapiFile.ReadFile("grpc-openapi.json")
	if err != nil {
		http.Error(w, "Failed to load OpenAPI file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func parseURL(url string) string {
	p1 := strings.Replace(url, "http://", "", -1)
	p1 = strings.Replace(url, "tcp://", "", -1)
	return p1
}

func setupGrpcSwagger(ctx *server.Context, router *mux.Router, svrCfg serverconfig.Config) *v3cdn.Handler {

	if !svrCfg.GRPC.Enable {
		return nil
	}

	settingsUI := make(map[string]string)
	settingsUI["requestInterceptor"] = `function(request) {
		if (request.loadSpec) {
			return request;
		}
		var url = window.location.protocol + '//'+ window.location.host;
		var method = request.url.substring(url.length);
		request.headers['Content-Type'] = 'application/json';
		return request;
	}`

	handler := v3cdn.NewHandlerWithConfig(swgui.Config{
		Title:       "Helios RPC",
		SwaggerJSON: "/docs/openapi.json",
		BasePath:    "/docs",
		SettingsUI:  settingsUI,
	})

	router.HandleFunc("/docs/openapi.json", serveOpenAPI).Methods("GET")
	router.HandleFunc("/docs", handler.ServeHTTP).Methods("GET")

	host, port, _ := net.SplitHostPort(parseURL(svrCfg.API.Address))
	docsURL := fmt.Sprintf("http://%s:%s/docs", host, port)

	ctx.Logger.Info("SWAGGER API GRPC Documentation", "url", docsURL)
	return handler
}
