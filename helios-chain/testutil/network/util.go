package network

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime/debug"
	"strings"

	"golang.org/x/sync/errgroup"

	"cosmossdk.io/log"
	cmtcfg "github.com/cometbft/cometbft/config"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/node"
	"github.com/cometbft/cometbft/p2p"
	pvm "github.com/cometbft/cometbft/privval"
	"github.com/cometbft/cometbft/proxy"
	"github.com/cometbft/cometbft/rpc/client/local"
	"github.com/cometbft/cometbft/types"
	cmttime "github.com/cometbft/cometbft/types/time"

	"github.com/ethereum/go-ethereum/ethclient"

	sdkserver "github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	servergrpc "github.com/cosmos/cosmos-sdk/server/grpc"
	servercmtlog "github.com/cosmos/cosmos-sdk/server/log"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"helios-core/helios-chain/server"
	evmtypes "helios-core/helios-chain/x/evm/types"
)

// Debug helper to print module state info
func debugPrintGenesisState(genesisState map[string]json.RawMessage) {
	fmt.Println("============= Genesis State Debug Info =============")
	if genesisState == nil {
		fmt.Println("WARNING: Genesis state is nil!")
		return
	}

	fmt.Printf("Total modules in genesis state: %d\n", len(genesisState))

	for module, data := range genesisState {
		fmt.Printf("Module: %-25s | Data length: %d bytes\n", module, len(data))
		if len(data) < 20 {
			fmt.Printf("  Content: %s\n", string(data))
		} else {
			fmt.Printf("  Content preview: %s...\n", string(data)[:20])
		}
	}

	// Check for critical modules
	criticalModules := []string{
		authtypes.ModuleName,
		banktypes.ModuleName,
		stakingtypes.ModuleName,
		govtypes.ModuleName,
		crisistypes.ModuleName,
		evmtypes.ModuleName,
	}

	fmt.Println("\nChecking critical modules:")
	for _, module := range criticalModules {
		if data, ok := genesisState[module]; ok {
			fmt.Printf("✓ Module %s is present with %d bytes\n", module, len(data))
		} else {
			fmt.Printf("✗ MISSING CRITICAL MODULE: %s\n", module)
		}
	}
	fmt.Println("===================================================")
}

func startInProcess(cfg Config, val *Validator) error {
	logger := val.Ctx.Logger
	cmtCfg := val.Ctx.Config
	cmtCfg.Instrumentation.Prometheus = false

	fmt.Printf("Starting in-process node for validator: %s\n", val.Moniker)

	if err := val.AppConfig.ValidateBasic(); err != nil {
		return fmt.Errorf("app config validation failed: %w", err)
	}

	nodeKey, err := p2p.LoadOrGenNodeKey(cmtCfg.NodeKeyFile())
	if err != nil {
		return fmt.Errorf("failed to load or generate node key: %w", err)
	}

	fmt.Println("Creating app instance...")
	app := cfg.AppConstructor(*val)
	val.app = app

	fmt.Println("Setting up node...")
	genDocProvider := server.GenDocProvider(cmtCfg)
	fmt.Println("Setting up node...1")
	cmtApp := sdkserver.NewCometABCIWrapper(app)
	fmt.Println("Setting up node...2")
	tmNode, err := node.NewNode(
		cmtCfg,
		pvm.LoadOrGenFilePV(cmtCfg.PrivValidatorKeyFile(), cmtCfg.PrivValidatorStateFile()),
		nodeKey,
		proxy.NewLocalClientCreator(cmtApp),
		genDocProvider,
		cmtcfg.DefaultDBProvider,
		node.DefaultMetricsProvider(cmtCfg.Instrumentation),
		servercmtlog.CometLoggerWrapper{Logger: logger.With("module", val.Moniker)},
	)
	fmt.Println("Setting up node...3")
	if err != nil {
		return fmt.Errorf("failed to create node: %w", err)
	}

	fmt.Println("Starting node...")
	if err := tmNode.Start(); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	val.tmNode = tmNode

	if val.RPCAddress != "" {
		val.RPCClient = local.New(tmNode)
	}

	// We'll need a RPC client if the validator exposes a gRPC or REST endpoint.
	if val.APIAddress != "" || val.AppConfig.GRPC.Enable {
		val.ClientCtx = val.ClientCtx.
			WithClient(val.RPCClient)

		// Add the tx service in the gRPC router.
		app.RegisterTxService(val.ClientCtx)

		// Add the tendermint queries service in the gRPC router.
		app.RegisterTendermintService(val.ClientCtx)
		app.RegisterNodeService(val.ClientCtx, val.AppConfig.Config)
	}

	ctx := context.Background()
	ctx, val.cancelFn = context.WithCancel(ctx)
	val.errGroup, ctx = errgroup.WithContext(ctx)

	if val.AppConfig.API.Enable && val.APIAddress != "" {
		fmt.Printf("Setting up API server at %s\n", val.APIAddress)
		apiSrv := api.New(val.ClientCtx, logger.With("module", "api-server"), val.grpc)
		app.RegisterAPIRoutes(apiSrv, val.AppConfig.API)

		val.errGroup.Go(func() error {
			return apiSrv.Start(ctx, val.AppConfig.Config)
		})

		val.api = apiSrv
	}

	if val.AppConfig.GRPC.Enable {
		fmt.Printf("Setting up gRPC server at %s\n", val.AppConfig.GRPC.Address)
		grpcSrv, err := servergrpc.NewGRPCServer(val.ClientCtx, app, val.AppConfig.GRPC)
		if err != nil {
			return fmt.Errorf("failed to create gRPC server: %w", err)
		}

		// Start the gRPC server in a goroutine. Note, the provided ctx will ensure
		// that the server is gracefully shut down.
		val.errGroup.Go(func() error {
			return servergrpc.StartGRPCServer(ctx, logger.With(log.ModuleKey, "grpc-server"), val.AppConfig.GRPC, grpcSrv)
		})

		val.grpc = grpcSrv
	}

	if val.AppConfig.JSONRPC.Enable && val.AppConfig.JSONRPC.Address != "" {
		fmt.Printf("Setting up JSON-RPC server at %s\n", val.AppConfig.JSONRPC.Address)
		if val.Ctx == nil || val.Ctx.Viper == nil {
			return fmt.Errorf("validator %s context is nil", val.Moniker)
		}

		tmEndpoint := "/websocket"
		tmRPCAddr := fmt.Sprintf("tcp://%s", val.AppConfig.GRPC.Address)

		val.jsonrpc, val.jsonrpcDone, err = server.StartJSONRPC(val.Ctx, val.ClientCtx, tmRPCAddr, tmEndpoint, val.AppConfig, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to start JSON-RPC: %w", err)
		}

		address := fmt.Sprintf("http://%s", val.AppConfig.JSONRPC.Address)

		val.JSONRPCClient, err = ethclient.Dial(address)
		if err != nil {
			return fmt.Errorf("failed to dial JSON-RPC at %s: %w", val.AppConfig.JSONRPC.Address, err)
		}
	}

	fmt.Printf("Node %s started successfully\n", val.Moniker)
	return nil
}

func collectGenFiles(cfg Config, vals []*Validator, outputDir string) error {
	fmt.Println("Collecting genesis files...")
	genTime := cmttime.Now()

	for i := 0; i < cfg.NumValidators; i++ {
		cmtCfg := vals[i].Ctx.Config

		nodeDir := filepath.Join(outputDir, vals[i].Moniker, "heliades")
		gentxsDir := filepath.Join(outputDir, "gentxs")

		cmtCfg.Moniker = vals[i].Moniker
		cmtCfg.SetRoot(nodeDir)

		fmt.Printf("Processing validator %d: %s\n", i, vals[i].Moniker)
		initCfg := genutiltypes.NewInitConfig(cfg.ChainID, gentxsDir, vals[i].NodeID, vals[i].PubKey)

		genFile := cmtCfg.GenesisFile()
		fmt.Printf("Reading genesis file from: %s\n", genFile)

		appGenesis, err := genutiltypes.AppGenesisFromFile(genFile)
		if err != nil {
			return fmt.Errorf("failed to read genesis file for validator %s: %w", vals[i].Moniker, err)
		}

		appState, err := genutil.GenAppStateFromConfig(cfg.Codec, cfg.TxConfig,
			cmtCfg, initCfg, appGenesis, banktypes.GenesisBalancesIterator{}, genutiltypes.DefaultMessageValidator, cfg.TxConfig.SigningContext().ValidatorAddressCodec())
		if err != nil {
			return fmt.Errorf("failed to generate app state for validator %s: %w", vals[i].Moniker, err)
		}

		// overwrite each validator's genesis file to have a canonical genesis time
		fmt.Printf("Exporting genesis file to: %s\n", genFile)
		if err := genutil.ExportGenesisFileWithTime(genFile, cfg.ChainID, nil, appState, genTime); err != nil {
			return fmt.Errorf("failed to export genesis file for validator %s: %w", vals[i].Moniker, err)
		}
	}

	fmt.Println("Genesis files collected successfully")
	return nil
}

func initGenFiles(cfg Config, genAccounts []authtypes.GenesisAccount, genBalances []banktypes.Balance, genFiles []string) error {
	fmt.Printf("\n========== INITIALIZING GENESIS FILES ==========\n")
	fmt.Printf("Chain ID: %s\n", cfg.ChainID)
	fmt.Printf("Number of validators: %d\n", cfg.NumValidators)
	fmt.Printf("Number of genesis accounts: %d\n", len(genAccounts))
	fmt.Printf("Number of genesis balances: %d\n", len(genBalances))

	// Debug print all genesis state
	debugPrintGenesisState(cfg.GenesisState)

	if cfg.GenesisState == nil {
		return fmt.Errorf("genesis state is nil - it must be initialized before calling initGenFiles")
	}

	// Process auth module state with robust error handling
	fmt.Println("\nProcessing auth module...")
	authData, ok := cfg.GenesisState[authtypes.ModuleName]
	if !ok || len(authData) == 0 {
		return fmt.Errorf("auth module genesis state is missing or empty")
	}

	// Debug the auth data
	previewLen := 30
	if len(authData) < previewLen {
		previewLen = len(authData)
	}
	fmt.Printf("Auth module data preview: %s\n", string(authData[:previewLen]))

	var authGenState authtypes.GenesisState

	// Use try/catch pattern for debugging panics
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during auth unmarshal: %v\n", r)
				fmt.Println("Stack trace:")
				fmt.Println(string(debug.Stack()))
			}
		}()
		cfg.Codec.MustUnmarshalJSON(authData, &authGenState)
	}()

	accounts, err := authtypes.PackAccounts(genAccounts)
	if err != nil {
		return fmt.Errorf("failed to pack accounts: %w", err)
	}

	authGenState.Accounts = append(authGenState.Accounts, accounts...)

	// Safely marshal JSON
	var authStateJSON json.RawMessage
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during auth marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		authStateJSON = cfg.Codec.MustMarshalJSON(&authGenState)
	}()

	if len(authStateJSON) == 0 {
		return fmt.Errorf("failed to marshal auth genesis state - result was empty")
	}

	cfg.GenesisState[authtypes.ModuleName] = authStateJSON
	fmt.Printf("Auth module processed successfully: %d accounts\n", len(authGenState.Accounts))

	// Process bank module
	fmt.Println("\nProcessing bank module...")
	bankData, ok := cfg.GenesisState[banktypes.ModuleName]
	if !ok || len(bankData) == 0 {
		return fmt.Errorf("bank module genesis state is missing or empty")
	}

	var bankGenState banktypes.GenesisState

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during bank unmarshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		cfg.Codec.MustUnmarshalJSON(bankData, &bankGenState)
	}()

	bankGenState.Balances = genBalances

	var bankStateJSON json.RawMessage
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during bank marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		bankStateJSON = cfg.Codec.MustMarshalJSON(&bankGenState)
	}()

	if len(bankStateJSON) == 0 {
		return fmt.Errorf("failed to marshal bank genesis state - result was empty")
	}

	cfg.GenesisState[banktypes.ModuleName] = bankStateJSON
	fmt.Printf("Bank module processed successfully: %d balances\n", len(bankGenState.Balances))

	// Process staking module state
	fmt.Println("\nProcessing staking module...")
	stakingData, ok := cfg.GenesisState[stakingtypes.ModuleName]
	if !ok || len(stakingData) == 0 {
		return fmt.Errorf("staking module genesis state is missing or empty")
	}

	var stakingGenState stakingtypes.GenesisState

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during staking unmarshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		cfg.Codec.MustUnmarshalJSON(stakingData, &stakingGenState)
	}()

	stakingGenState.Params.BondDenom = cfg.BondDenom

	var stakingStateJSON json.RawMessage
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during staking marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		stakingStateJSON = cfg.Codec.MustMarshalJSON(&stakingGenState)
	}()

	if len(stakingStateJSON) == 0 {
		return fmt.Errorf("failed to marshal staking genesis state - result was empty")
	}

	cfg.GenesisState[stakingtypes.ModuleName] = stakingStateJSON
	fmt.Printf("Staking module processed successfully, bond denom set to: %s\n", cfg.BondDenom)

	// Process gov module state
	fmt.Println("\nProcessing gov module...")
	govData, ok := cfg.GenesisState[govtypes.ModuleName]
	if !ok || len(govData) == 0 {
		return fmt.Errorf("gov module genesis state is missing or empty")
	}

	var govGenState govv1.GenesisState

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during gov unmarshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
				// When a panic occurs here, print more details about the data
				if len(govData) > 0 {
					fmt.Printf("Gov data starts with: %s\n", string(govData[:min(100, len(govData))]))
				}
			}
		}()
		cfg.Codec.MustUnmarshalJSON(govData, &govGenState)
	}()

	// Ensure we have min deposit params before accessing
	if len(govGenState.Params.MinDeposit) > 0 {
		govGenState.Params.MinDeposit[0].Denom = cfg.BondDenom
	} else {
		fmt.Printf("WARNING: Gov module min deposit params are empty\n")
	}

	// Check expedited min deposit as well
	if len(govGenState.Params.ExpeditedMinDeposit) > 0 {
		govGenState.Params.ExpeditedMinDeposit[0].Denom = cfg.BondDenom
	} else {
		fmt.Printf("WARNING: Gov module expedited min deposit params are empty\n")
	}

	var govStateJSON json.RawMessage
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during gov marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		govStateJSON = cfg.Codec.MustMarshalJSON(&govGenState)
	}()

	if len(govStateJSON) == 0 {
		return fmt.Errorf("failed to marshal gov genesis state - result was empty")
	}

	cfg.GenesisState[govtypes.ModuleName] = govStateJSON
	fmt.Printf("Gov module processed successfully\n")

	// Process crisis module state
	fmt.Println("\nProcessing crisis module...")
	crisisData, ok := cfg.GenesisState[crisistypes.ModuleName]
	if !ok || len(crisisData) == 0 {
		return fmt.Errorf("crisis module genesis state is missing or empty")
	}

	var crisisGenState crisistypes.GenesisState

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during crisis unmarshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		cfg.Codec.MustUnmarshalJSON(crisisData, &crisisGenState)
	}()

	crisisGenState.ConstantFee.Denom = cfg.BondDenom

	var crisisStateJSON json.RawMessage
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during crisis marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		crisisStateJSON = cfg.Codec.MustMarshalJSON(&crisisGenState)
	}()

	if len(crisisStateJSON) == 0 {
		return fmt.Errorf("failed to marshal crisis genesis state - result was empty")
	}

	cfg.GenesisState[crisistypes.ModuleName] = crisisStateJSON
	fmt.Printf("Crisis module processed successfully\n")

	// Process evm module state - with additional safety checks
	fmt.Println("\nProcessing evm module...")
	evmData, ok := cfg.GenesisState[evmtypes.ModuleName]
	if !ok || len(evmData) == 0 {
		return fmt.Errorf("evm module genesis state is missing or empty")
	}

	var evmGenState evmtypes.GenesisState

	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during evm unmarshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
				fmt.Printf("EVM data preview: %s\n", string(evmData[:min(100, len(evmData))]))
			}
		}()
		cfg.Codec.MustUnmarshalJSON(evmData, &evmGenState)
	}()

	// Don't modify EVM state, just verify we could unmarshal it
	fmt.Printf("EVM module processed successfully\n")

	// Final marshaling of the complete genesis state
	fmt.Println("\nFinalizing genesis state...")

	// Marshal with standard json library for better error messages if there's an issue
	var appGenStateJSON []byte
	func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("PANIC during final genesis marshal: %v\n", r)
				fmt.Println(string(debug.Stack()))
			}
		}()
		appGenStateJSON, err = json.MarshalIndent(cfg.GenesisState, "", "  ")
		if err != nil {
			fmt.Printf("ERROR marshaling app genesis state: %v\n", err)
		}
	}()

	if err != nil {
		return fmt.Errorf("failed to marshal app genesis state: %w", err)
	}

	if len(appGenStateJSON) == 0 {
		return fmt.Errorf("marshaled app genesis state is empty")
	}

	genDoc := types.GenesisDoc{
		ChainID:    cfg.ChainID,
		AppState:   appGenStateJSON,
		Validators: nil,
	}

	// Generate empty genesis files for each validator and save
	fmt.Printf("\nGenerating genesis files for %d validators\n", cfg.NumValidators)
	for i := 0; i < cfg.NumValidators; i++ {
		fmt.Printf("Saving genesis file %d: %s\n", i, genFiles[i])
		if err := genDoc.SaveAs(genFiles[i]); err != nil {
			return fmt.Errorf("failed to save genesis file for validator %d: %w", i, err)
		}
	}

	fmt.Println("\nGenesis files initialized successfully!")
	return nil
}

// Helper function for min since Go < 1.21 doesn't have it in stdlib
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func WriteFile(name string, dir string, contents []byte) error {
	file := filepath.Join(dir, name)

	err := tmos.EnsureDir(dir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to ensure directory exists: %w", err)
	}

	fmt.Printf("Writing file: %s\n", file)
	return tmos.WriteFile(file, contents, 0o644)
}

// Additional debug helpers

// DumpGenesisFile prints the contents of a genesis file for debugging
func DumpGenesisFile(path string) error {
	fmt.Printf("Dumping genesis file: %s\n", path)

	// Read the genesis file
	genDoc, err := types.GenesisDocFromFile(path)
	if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	fmt.Printf("Chain ID: %s\n", genDoc.ChainID)
	fmt.Printf("Genesis time: %s\n", genDoc.GenesisTime)
	fmt.Printf("App state size: %d bytes\n", len(genDoc.AppState))

	// Try to parse app state as JSON
	var appState map[string]json.RawMessage
	if err := json.Unmarshal(genDoc.AppState, &appState); err != nil {
		return fmt.Errorf("failed to unmarshal app state: %w", err)
	}

	// Print modules in app state
	modules := make([]string, 0, len(appState))
	for module := range appState {
		modules = append(modules, module)
	}
	fmt.Printf("Modules in app state: %s\n", strings.Join(modules, ", "))

	return nil
}

// VerifyChainConfig validates the chain configuration for common issues
func VerifyChainConfig(cfg Config) []string {
	issues := []string{}

	if cfg.ChainID == "" {
		issues = append(issues, "Chain ID is empty")
	}

	if cfg.Codec == nil {
		issues = append(issues, "Codec is nil")
	}

	if cfg.TxConfig == nil {
		issues = append(issues, "TxConfig is nil")
	}

	if cfg.GenesisState == nil {
		issues = append(issues, "GenesisState is nil")
	}

	if cfg.BondDenom == "" {
		issues = append(issues, "BondDenom is empty")
	}

	return issues
}
