package network

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtrand "github.com/cometbft/cometbft/libs/rand"
	"github.com/cometbft/cometbft/node"
	cmtclient "github.com/cometbft/cometbft/rpc/client"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	"helios-core/helios-chain/app"
	"helios-core/helios-chain/crypto/hd"

	pruningtypes "cosmossdk.io/store/pruning/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	srvconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	simutils "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"helios-core/helios-chain/server/config"
	evmostypes "helios-core/helios-chain/types"
)

// package-wide network lock to only allow one test network at a time
var (
	lock = new(sync.Mutex)
	// portPool = make(chan string, 20000)
)

// AppConstructor defines a function which accepts a network configuration and
// creates an ABCI Application to provide to Tendermint.
type AppConstructor = func(val Validator) servertypes.Application

// Config defines the necessary configuration used to bootstrap and start an
// in-process local testing network.
type Config struct {
	KeyringOptions    []keyring.Option // keyring configuration options
	Codec             codec.Codec
	LegacyAmino       *codec.LegacyAmino // TODO: Remove!
	InterfaceRegistry codectypes.InterfaceRegistry
	TxConfig          client.TxConfig
	AccountRetriever  client.AccountRetriever
	AppConstructor    AppConstructor          // the ABCI application constructor
	GenesisState      evmostypes.GenesisState // custom gensis state to provide
	TimeoutCommit     time.Duration           // the consensus commitment timeout
	AccountTokens     math.Int                // the amount of unique validator tokens (e.g. 1000node0)
	StakingTokens     math.Int                // the amount of tokens each validator has available to stake
	BondedTokens      math.Int                // the amount of tokens each validator stakes
	NumValidators     int                     // the total number of validators to create and bond
	ChainID           string                  // the network chain-id
	BondDenom         string                  // the staking bond denomination
	MinGasPrices      string                  // the minimum gas prices each validator will accept
	PruningStrategy   string                  // the pruning strategy each validator will have
	SigningAlgo       string                  // signing algorithm for keys
	RPCAddress        string                  // RPC listen address (including port)
	JSONRPCAddress    string                  // JSON-RPC listen address (including port)
	APIAddress        string                  // REST API listen address (including port)
	GRPCAddress       string                  // GRPC server listen address (including port)
	EnableTMLogging   bool                    // enable Tendermint logging to STDOUT
	CleanupDir        bool                    // remove base temporary directory during cleanup
	PrintMnemonic     bool                    // print the mnemonic of first validator as log output for testing
}

// DefaultConfig returns a sane default configuration suitable for nearly all
// testing requirements.
func DefaultConfig() Config {
	chainID := fmt.Sprintf("%d-1", cmtrand.Int63n(9999999999999)+1)
	dir, err := os.MkdirTemp("", "simapp")
	if err != nil {
		panic(fmt.Sprintf("failed creating temporary directory: %v", err))
	}
	defer os.RemoveAll(dir)
	app := app.NewHeliosApp(log.NewNopLogger(), dbm.NewMemDB(), map[string]dbm.DB{
		"hyperion": dbm.NewMemDB(),
		"chronos":  dbm.NewMemDB(),
	}, nil, true, simutils.NewAppOptionsWithFlagHome(dir), baseapp.SetChainID(chainID))
	return Config{
		Codec:             app.AppCodec(),
		TxConfig:          app.GetTxConfig(),
		LegacyAmino:       app.LegacyAmino(),
		InterfaceRegistry: app.InterfaceRegistry(),
		AccountRetriever:  authtypes.AccountRetriever{},
		AppConstructor:    NewAppConstructor(chainID),
		GenesisState:      app.DefaultGenesis(),
		// GenesisState:    nil, //TODO: Fix diffs and set back default
		TimeoutCommit:   3 * time.Second,
		ChainID:         chainID,
		NumValidators:   2,
		BondDenom:       evmostypes.BaseDenom,
		MinGasPrices:    fmt.Sprintf("0.000006%s", evmostypes.BaseDenom),
		AccountTokens:   sdk.TokensFromConsensusPower(1000000000000000000, evmostypes.PowerReduction),
		StakingTokens:   sdk.TokensFromConsensusPower(500000000000000000, evmostypes.PowerReduction),
		BondedTokens:    sdk.TokensFromConsensusPower(100000000000000000, evmostypes.PowerReduction),
		PruningStrategy: pruningtypes.PruningOptionNothing,
		CleanupDir:      true,
		SigningAlgo:     string(hd.EthSecp256k1Type),
		KeyringOptions:  []keyring.Option{hd.EthSecp256k1Option()},
		PrintMnemonic:   false,
	}
}

// NewAppConstructor returns a new Helios AppConstructor
func NewAppConstructor(chainID string) AppConstructor {
	return func(val Validator) servertypes.Application {
		return app.NewHeliosApp(
			val.Ctx.Logger, dbm.NewMemDB(), map[string]dbm.DB{
				"hyperion": dbm.NewMemDB(),
				"chronos":  dbm.NewMemDB(),
			}, nil, true,
			simutils.NewAppOptionsWithFlagHome(val.Ctx.Config.RootDir),
			baseapp.SetPruning(pruningtypes.NewPruningOptionsFromString(val.AppConfig.Pruning)),
			baseapp.SetMinGasPrices(val.AppConfig.MinGasPrices),
			baseapp.SetChainID(chainID),
		)
	}
}

type (
	// Network defines a local in-process testing network using SimApp. It can be
	// configured to start any number of validators, each with its own RPC and API
	// clients. Typically, this test network would be used in client and integration
	// testing where user input is expected.
	//
	// Note, due to Tendermint constraints in regards to RPC functionality, there
	// may only be one test network running at a time. Thus, any caller must be
	// sure to Cleanup after testing is finished in order to allow other tests
	// to create networks. In addition, only the first validator will have a valid
	// RPC and API server/client.
	Network struct {
		Logger     Logger
		BaseDir    string
		Validators []*Validator

		Config Config
	}

	// Validator defines an in-process Tendermint validator node. Through this object,
	// a client can make RPC and API calls and interact with any client command
	// or handler.
	Validator struct {
		AppConfig     *config.Config
		ClientCtx     client.Context
		Ctx           *server.Context
		Dir           string
		NodeID        string
		PubKey        cryptotypes.PubKey
		Moniker       string
		APIAddress    string
		RPCAddress    string
		P2PAddress    string
		Address       sdk.AccAddress
		ValAddress    sdk.ValAddress
		RPCClient     cmtclient.Client
		JSONRPCClient *ethclient.Client

		app         servertypes.Application
		tmNode      *node.Node
		api         *api.Server
		grpc        *grpc.Server
		grpcWeb     *http.Server
		jsonrpc     *http.Server
		cdn         *http.Server
		cdnDone     chan struct{}
		jsonrpcDone chan struct{}
		errGroup    *errgroup.Group
		cancelFn    context.CancelFunc
	}
)

// Logger is a network logger interface that exposes testnet-level Log() methods for an in-process testing network
// This is not to be confused with logging that may happen at an individual node or validator level
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
}

var (
	_ Logger = (*testing.T)(nil)
	_ Logger = (*CLILogger)(nil)
)

type CLILogger struct {
	cmd *cobra.Command
}

func (s CLILogger) Log(args ...interface{}) {
	s.cmd.Println(args...)
}

func (s CLILogger) Logf(format string, args ...interface{}) {
	s.cmd.Printf(format, args...)
}

func NewCLILogger(cmd *cobra.Command) CLILogger {
	return CLILogger{cmd}
}

// New creates a new Network for integration tests or in-process testnets run via the CLI
func New(l Logger, baseDir string, cfg Config) (*Network, error) {
	// only one caller/test can create and use a network at a time
	l.Log("acquiring test network lock")
	lock.Lock()

	if !evmostypes.IsValidChainID(cfg.ChainID) {
		return nil, fmt.Errorf("invalid chain-id: %s", cfg.ChainID)
	}

	network := &Network{
		Logger:     l,
		BaseDir:    baseDir,
		Validators: make([]*Validator, cfg.NumValidators),
		Config:     cfg,
	}

	l.Logf("preparing test network with chain-id \"%s\"\n", cfg.ChainID)

	monikers := make([]string, cfg.NumValidators)
	nodeIDs := make([]string, cfg.NumValidators)
	valPubKeys := make([]cryptotypes.PubKey, cfg.NumValidators)

	var (
		genAccounts []authtypes.GenesisAccount
		genBalances []banktypes.Balance
		genFiles    []string
	)

	buf := bufio.NewReader(os.Stdin)

	// generate private keys, node IDs, and initial transactions
	for i := 0; i < cfg.NumValidators; i++ {
		appCfg := config.DefaultConfig()
		appCfg.Pruning = cfg.PruningStrategy
		appCfg.MinGasPrices = cfg.MinGasPrices
		appCfg.API.Enable = true
		appCfg.API.Swagger = false
		appCfg.Telemetry.Enabled = false
		appCfg.Telemetry.GlobalLabels = [][]string{{"chain_id", cfg.ChainID}}

		ctx := server.NewDefaultContext()
		cmtCfg := ctx.Config
		cmtCfg.Consensus.TimeoutCommit = cfg.TimeoutCommit

		// Only allow the first validator to expose an RPC, API and gRPC
		// server/client due to Tendermint in-process constraints.
		apiAddr := ""
		cmtCfg.RPC.ListenAddress = ""
		appCfg.GRPC.Enable = false
		appCfg.GRPCWeb.Enable = false
		appCfg.JSONRPC.Enable = false
		apiListenAddr := ""
		if i == 0 {
			if cfg.APIAddress != "" {
				apiListenAddr = cfg.APIAddress
			} else {
				port := 10317 - i
				apiListenAddr = fmt.Sprintf("tcp://0.0.0.0:%s", strconv.Itoa(port))
			}

			appCfg.API.Address = apiListenAddr
			apiURL, err := url.Parse(apiListenAddr)
			if err != nil {
				return nil, err
			}
			apiAddr = fmt.Sprintf("http://%s:%s", apiURL.Hostname(), apiURL.Port())

			if cfg.RPCAddress != "" {
				cmtCfg.RPC.ListenAddress = cfg.RPCAddress
			} else {
				port := 15000 - i
				cmtCfg.RPC.ListenAddress = fmt.Sprintf("tcp://0.0.0.0:%s", strconv.Itoa(port))
			}

			if cfg.GRPCAddress != "" {
				appCfg.GRPC.Address = cfg.GRPCAddress
			} else {
				port := 9090 - i
				appCfg.GRPC.Address = fmt.Sprintf("0.0.0.0:%s", strconv.Itoa(port))
			}
			appCfg.GRPC.Enable = true
			appCfg.GRPCWeb.Enable = true

			if cfg.JSONRPCAddress != "" {
				appCfg.JSONRPC.Address = cfg.JSONRPCAddress
			} else {
				port := 8545 - i
				appCfg.JSONRPC.Address = fmt.Sprintf("0.0.0.0:%s", strconv.Itoa(port))
			}
			appCfg.JSONRPC.Enable = true
			appCfg.JSONRPC.API = config.GetAPINamespaces()
		}

		logger := log.NewNopLogger()
		if cfg.EnableTMLogging {
			logger = log.NewLogger(os.Stdout)
		}

		ctx.Logger = logger

		nodeDirName := fmt.Sprintf("node%d", i)
		nodeDir := filepath.Join(network.BaseDir, nodeDirName, "heliades")
		clientDir := filepath.Join(network.BaseDir, nodeDirName, "heliadescli")
		gentxsDir := filepath.Join(network.BaseDir, "gentxs")

		err := os.MkdirAll(filepath.Join(nodeDir, "config"), 0o750)
		if err != nil {
			return nil, err
		}

		err = os.MkdirAll(clientDir, 0o750)
		if err != nil {
			return nil, err
		}

		cmtCfg.SetRoot(nodeDir)
		cmtCfg.Moniker = nodeDirName
		monikers[i] = nodeDirName

		port := i + 10100
		l.Log("proxyApp", port)
		proxyAddr := fmt.Sprintf("tcp://0.0.0.0:%s", strconv.Itoa(port))
		cmtCfg.ProxyApp = proxyAddr

		port = i + 18500
		p2pAddr := "tcp://0.0.0.0:18555" //fmt.Sprintf("tcp://0.0.0.0:%s", port)
		cmtCfg.P2P.ListenAddress = p2pAddr
		cmtCfg.P2P.AddrBookStrict = false
		cmtCfg.P2P.AllowDuplicateIP = true

		nodeID, pubKey, err := genutil.InitializeNodeValidatorFiles(cmtCfg)
		if err != nil {
			return nil, err
		}
		nodeIDs[i] = nodeID
		valPubKeys[i] = pubKey

		kb, err := keyring.New(sdk.KeyringServiceName(), keyring.BackendTest, clientDir, buf, cfg.Codec, cfg.KeyringOptions...)
		if err != nil {
			return nil, err
		}

		keyringAlgos, _ := kb.SupportedAlgorithms()
		algo, err := keyring.NewSigningAlgoFromString(cfg.SigningAlgo, keyringAlgos)
		if err != nil {
			return nil, err
		}

		addr, secret, err := testutil.GenerateSaveCoinKey(kb, nodeDirName, "", true, algo)
		if err != nil {
			return nil, err
		}

		// if PrintMnemonic is set to true, we print the first validator node's secret to the network's logger
		// for debugging and manual testing
		if cfg.PrintMnemonic && i == 0 {
			printMnemonic(l, secret)
		}

		info := map[string]string{"secret": secret}
		infoBz, err := json.Marshal(info)
		if err != nil {
			return nil, err
		}

		// save private key seed words
		err = WriteFile(fmt.Sprintf("%v.json", "key_seed"), clientDir, infoBz)
		if err != nil {
			return nil, err
		}

		balances := sdk.NewCoins(
			sdk.NewCoin(fmt.Sprintf("%stoken", nodeDirName), cfg.AccountTokens),
			sdk.NewCoin(cfg.BondDenom, cfg.StakingTokens),
		)

		genFiles = append(genFiles, cmtCfg.GenesisFile())
		genBalances = append(genBalances, banktypes.Balance{Address: addr.String(), Coins: balances.Sort()})
		genAccounts = append(genAccounts, authtypes.NewBaseAccount(addr, nil, 0, 0))

		commission, err := math.LegacyNewDecFromStr("0.5")
		if err != nil {
			return nil, err
		}

		createValMsg, err := stakingtypes.NewMsgCreateValidator(
			sdk.ValAddress(addr).String(),
			valPubKeys[i],
			sdk.NewCoin(cfg.BondDenom, cfg.BondedTokens),
			stakingtypes.NewDescription(nodeDirName, "", "", "", ""),
			stakingtypes.NewCommissionRates(commission, math.LegacyOneDec(), math.LegacyOneDec()),
			math.OneInt(),
		)
		if err != nil {
			return nil, err
		}

		p2pURL, err := url.Parse(p2pAddr)
		if err != nil {
			return nil, err
		}

		memo := fmt.Sprintf("%s@%s:%s", nodeIDs[i], p2pURL.Hostname(), p2pURL.Port())
		fee := sdk.NewCoins(sdk.NewCoin(cfg.BondDenom, math.NewInt(0)))
		txBuilder := cfg.TxConfig.NewTxBuilder()
		err = txBuilder.SetMsgs(createValMsg)
		if err != nil {
			return nil, err
		}
		txBuilder.SetFeeAmount(fee)    // Arbitrary fee
		txBuilder.SetGasLimit(1000000) // Need at least 100386
		txBuilder.SetMemo(memo)

		txFactory := tx.Factory{}
		txFactory = txFactory.
			WithChainID(cfg.ChainID).
			WithMemo(memo).
			WithKeybase(kb).
			WithTxConfig(cfg.TxConfig)

		if err := tx.Sign(context.Background(), txFactory, nodeDirName, txBuilder, true); err != nil {
			return nil, err
		}

		txBz, err := cfg.TxConfig.TxJSONEncoder()(txBuilder.GetTx())
		if err != nil {
			return nil, err
		}

		if err := WriteFile(fmt.Sprintf("%v.json", nodeDirName), gentxsDir, txBz); err != nil {
			return nil, err
		}

		customAppTemplate, _ := config.AppConfig(evmostypes.BaseDenom)
		srvconfig.SetConfigTemplate(customAppTemplate)
		srvconfig.WriteConfigFile(filepath.Join(nodeDir, "config/app.toml"), appCfg)

		ctx.Viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		ctx.Viper.SetConfigFile(filepath.Join(nodeDir, "config/app.toml"))
		err = ctx.Viper.ReadInConfig()
		if err != nil {
			return nil, err
		}

		clientCtx := client.Context{}.
			WithKeyringDir(clientDir).
			WithKeyring(kb).
			WithHomeDir(cmtCfg.RootDir).
			WithChainID(cfg.ChainID).
			WithInterfaceRegistry(cfg.InterfaceRegistry).
			WithCodec(cfg.Codec).
			WithLegacyAmino(cfg.LegacyAmino).
			WithTxConfig(cfg.TxConfig).
			WithAccountRetriever(cfg.AccountRetriever)

		network.Validators[i] = &Validator{
			AppConfig:  appCfg,
			ClientCtx:  clientCtx,
			Ctx:        ctx,
			Dir:        filepath.Join(network.BaseDir, nodeDirName),
			NodeID:     nodeID,
			PubKey:     pubKey,
			Moniker:    nodeDirName,
			RPCAddress: cmtCfg.RPC.ListenAddress,
			P2PAddress: cmtCfg.P2P.ListenAddress,
			APIAddress: apiAddr,
			Address:    addr,
			ValAddress: sdk.ValAddress(addr),
		}
	}

	err := initGenFiles(cfg, genAccounts, genBalances, genFiles)
	if err != nil {
		return nil, err
	}
	err = collectGenFiles(cfg, network.Validators, network.BaseDir)
	if err != nil {
		return nil, err
	}

	l.Log("starting test network...")
	for _, v := range network.Validators {
		err := startInProcess(cfg, v)
		if err != nil {
			return nil, err
		}
	}

	l.Log("started test network")

	// Ensure we cleanup incase any test was abruptly halted (e.g. SIGINT) as any
	// defer in a test would not be called.
	trapSignal(network.Cleanup)

	return network, nil
}

// LatestHeight returns the latest height of the network or an error if the
// query fails or no validators exist.
func (n *Network) LatestHeight() (int64, error) {
	if len(n.Validators) == 0 {
		return 0, errors.New("no validators available")
	}

	status, err := n.Validators[0].RPCClient.Status(context.Background())
	if err != nil {
		return 0, err
	}

	return status.SyncInfo.LatestBlockHeight, nil
}

// WaitForHeight performs a blocking check where it waits for a block to be
// committed after a given block. If that height is not reached within a timeout,
// an error is returned. Regardless, the latest height queried is returned.
func (n *Network) WaitForHeight(h int64) (int64, error) {
	return n.WaitForHeightWithTimeout(h, 10*time.Second)
}

// WaitForHeightWithTimeout is the same as WaitForHeight except the caller can
// provide a custom timeout.
func (n *Network) WaitForHeightWithTimeout(h int64, t time.Duration) (int64, error) {
	ticker := time.NewTicker(time.Second)
	timeout := time.After(t)

	if len(n.Validators) == 0 {
		return 0, errors.New("no validators available")
	}

	var latestHeight int64
	val := n.Validators[0]

	for {
		select {
		case <-timeout:
			ticker.Stop()
			return latestHeight, errors.New("timeout exceeded waiting for block")
		case <-ticker.C:
			status, err := val.RPCClient.Status(context.Background())
			if err == nil && status != nil {
				latestHeight = status.SyncInfo.LatestBlockHeight
				if latestHeight >= h {
					return latestHeight, nil
				}
			}
		}
	}
}

// WaitForNextBlock waits for the next block to be committed, returning an error
// upon failure.
func (n *Network) WaitForNextBlock() error {
	lastBlock, err := n.LatestHeight()
	if err != nil {
		return err
	}

	_, err = n.WaitForHeight(lastBlock + 1)
	if err != nil {
		return err
	}

	return err
}

// Cleanup removes the root testing (temporary) directory and stops both the
// Tendermint and API services. It allows other callers to create and start
// test networks. This method must be called when a test is finished, typically
// in a defer.
func (n *Network) Cleanup() {
	defer func() {
		lock.Unlock()
		n.Logger.Log("released test network lock")
	}()

	n.Logger.Log("cleaning up test network...")

	for _, v := range n.Validators {
		if v.tmNode != nil && v.tmNode.IsRunning() {
			_ = v.tmNode.Stop()
		}

		if v.api != nil {
			_ = v.api.Close()
		}

		if v.grpc != nil {
			v.grpc.Stop()
			if v.grpcWeb != nil {
				_ = v.grpcWeb.Close()
			}
		}

		if v.jsonrpc != nil {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()

			if err := v.jsonrpc.Shutdown(shutdownCtx); err != nil {
				v.tmNode.Logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
			} else {
				v.tmNode.Logger.Info("HTTP server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-v.jsonrpcDone:
				}
			}
		}

		if v.cdn != nil {
			shutdownCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()

			if err := v.cdn.Shutdown(shutdownCtx); err != nil {
				v.tmNode.Logger.Error("HTTP server shutdown produced a warning", "error", err.Error())
			} else {
				v.tmNode.Logger.Info("HTTP server shut down, waiting 5 sec")
				select {
				case <-time.Tick(5 * time.Second):
				case <-v.cdnDone:
				}
			}
		}
	}

	if n.Config.CleanupDir {
		_ = os.RemoveAll(n.BaseDir)
	}

	n.Logger.Log("finished cleaning up test network")
}

// printMnemonic prints a provided mnemonic seed phrase on a network logger
// for debugging and manual testing
func printMnemonic(l Logger, secret string) {
	lines := []string{
		"THIS MNEMONIC IS FOR TESTING PURPOSES ONLY",
		"DO NOT USE IN PRODUCTION",
		"",
		strings.Join(strings.Fields(secret)[0:8], " "),
		strings.Join(strings.Fields(secret)[8:16], " "),
		strings.Join(strings.Fields(secret)[16:24], " "),
	}

	lineLengths := make([]int, len(lines))
	for i, line := range lines {
		lineLengths[i] = len(line)
	}

	maxLineLength := 0
	for _, lineLen := range lineLengths {
		if lineLen > maxLineLength {
			maxLineLength = lineLen
		}
	}

	l.Log("\n")
	l.Log(strings.Repeat("+", maxLineLength+8))
	for _, line := range lines {
		l.Logf("++  %s  ++\n", centerText(line, maxLineLength))
	}
	l.Log(strings.Repeat("+", maxLineLength+8))
	l.Log("\n")
}

// centerText centers text across a fixed width, filling either side with whitespace buffers
func centerText(text string, width int) string {
	textLen := len(text)
	leftBuffer := strings.Repeat(" ", (width-textLen)/2)
	rightBuffer := strings.Repeat(" ", (width-textLen)/2+(width-textLen)%2)

	return fmt.Sprintf("%s%s%s", leftBuffer, text, rightBuffer)
}

// trapSignal traps SIGINT and SIGTERM and calls os.Exit once a signal is received.
func trapSignal(cleanupFunc func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs

		if cleanupFunc != nil {
			cleanupFunc()
		}
		exitCode := 128

		switch sig {
		case syscall.SIGINT:
			exitCode += int(syscall.SIGINT)
		case syscall.SIGTERM:
			exitCode += int(syscall.SIGTERM)
		}

		os.Exit(exitCode)
	}()
}
