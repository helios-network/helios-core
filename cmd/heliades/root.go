// Copyright Tharsis Labs Ltd.(Evmos)
// SPDX-License-Identifier:ENCL-1.0(https://github.com/helios/helios/blob/main/LICENSE)

package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"cosmossdk.io/log"
	cmtcfg "github.com/cometbft/cometbft/config"
	cmtcli "github.com/cometbft/cometbft/libs/cli"
	dbm "github.com/cosmos/cosmos-db"

	heliostypes "helios-core/helios-chain/types"

	"cosmossdk.io/store"
	"cosmossdk.io/store/snapshots"
	snapshottypes "cosmossdk.io/store/snapshots/types"
	storetypes "cosmossdk.io/store/types"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	clientcfg "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	sdkserver "github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	txmodule "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	rosettaCmd "github.com/cosmos/rosetta/cmd"

	chainclient "helios-core/helios-chain/client"
	"helios-core/helios-chain/client/block"
	"helios-core/helios-chain/client/debug"
	heliosserver "helios-core/helios-chain/server"
	servercfg "helios-core/helios-chain/server/config"
	srvflags "helios-core/helios-chain/server/flags"

	"helios-core/helios-chain/crypto/hd"

	"helios-core/helios-chain/app"
	helioskr "helios-core/helios-chain/crypto/keyring"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmcli "github.com/CosmWasm/wasmd/x/wasm/client/cli"
)

const EnvPrefix = "helios"

type emptyAppOptions struct{}

func (ao emptyAppOptions) Get(_ string) interface{} { return nil }

// NewRootCmd creates a new root command for heliades. It is called once in the
// main function.
func NewRootCmd() (*cobra.Command, sdktestutil.TestEncodingConfig) {
	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	// and the CLI options for the modules
	// add keyring to autocli opts
	tempApp := app.NewHeliosApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil, true,
		emptyAppOptions{},
	)
	encodingConfig := sdktestutil.TestEncodingConfig{
		InterfaceRegistry: tempApp.InterfaceRegistry(),
		Codec:             tempApp.AppCodec(),
		TxConfig:          tempApp.GetTxConfig(),
		Amino:             tempApp.LegacyAmino(),
	}
	initClientCtx := client.Context{}.
		WithCodec(encodingConfig.Codec).
		WithInterfaceRegistry(encodingConfig.InterfaceRegistry).
		WithTxConfig(encodingConfig.TxConfig).
		WithLegacyAmino(encodingConfig.Amino).
		WithInput(os.Stdin).
		WithKeyringOptions(hd.EthSecp256k1Option()).
		WithAccountRetriever(types.AccountRetriever{}).
		WithBroadcastMode(flags.BroadcastSync).
		WithHomeDir(app.DefaultNodeHome).
		WithKeyringOptions(helioskr.Option()).
		WithViper(EnvPrefix).
		WithLedgerHasProtobuf(true)

	rootCmd := &cobra.Command{
		Use:   "heliades",
		Short: "Helios Daemon",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			// set the default command outputs
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = clientcfg.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				enabledSignModes := append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL) //nolint:gocritic
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           enabledSignModes,
					TextualCoinMetadataQueryFn: txmodule.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txConfig, err := tx.NewTxConfigWithOptions(
					initClientCtx.Codec,
					txConfigOpts,
				)
				if err != nil {
					return err
				}

				initClientCtx = initClientCtx.WithTxConfig(txConfig)
			}

			if err := client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			// If the chainID was set in a flag or in the client.toml file, we can init the config here.
			// NOTE: if it is not set, it will default to "" and the function will be a no-op call.
			if err := app.InitializeAppConfiguration(initClientCtx.ChainID); err != nil {
				return fmt.Errorf("failed to initialize app configuration: %w", err)
			}

			// override the app and tendermint configuration
			customAppTemplate, customAppConfig := initAppConfig()
			customTMConfig := initTendermintConfig()

			return sdkserver.InterceptConfigsPreRunHandler(cmd, customAppTemplate, customAppConfig, customTMConfig)
		},
	}

	cfg := sdk.GetConfig()
	cfg.Seal()

	a := appCreator{encodingConfig}
	rootCmd.AddCommand(
		genutilcli.InitCmd(tempApp.BasicModuleManager, app.DefaultNodeHome),
		// heliosclient.ValidateChainID(
		// 	InitCmd(tempApp.BasicModuleManager, app.DefaultNodeHome),
		// ),
		genutilcli.CollectGenTxsCmd(banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			genutiltypes.DefaultMessageValidator,
			tempApp.GetTxConfig().SigningContext().ValidatorAddressCodec(),
		),
		MigrateGenesisCmd(),
		genutilcli.GenTxCmd(
			tempApp.BasicModuleManager, tempApp.GetTxConfig(),
			banktypes.GenesisBalancesIterator{},
			app.DefaultNodeHome,
			tempApp.GetTxConfig().SigningContext().ValidatorAddressCodec(),
		),
		genutilcli.ValidateGenesisCmd(tempApp.BasicModuleManager),
		AddGenesisAccountCmd(app.DefaultNodeHome),
		cmtcli.NewCompletionCmd(rootCmd, true),
		NewTestnetCmd(tempApp.BasicModuleManager, banktypes.GenesisBalancesIterator{}),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(a.newApp, app.DefaultNodeHome),
		db.Cmd(a.newApp, app.DefaultNodeHome),
		db.BlockstoreCmd(a.newApp, app.DefaultNodeHome),
		db.StatedbCmd(a.newApp, app.DefaultNodeHome),
		snapshot.Cmd(a.newApp),
		block.Cmd(),
	)

	changeSetCmd := ChangeSetCmd()
	if changeSetCmd != nil {
		rootCmd.AddCommand(changeSetCmd)
	}

	heliosserver.AddCommands(
		rootCmd,
		heliosserver.NewDefaultStartOptions(a.newApp, app.DefaultNodeHome),
		a.appExport,
		addModuleInitFlags,
	)

	wasmcli.ExtendUnsafeResetAllCmd(rootCmd)

	// add keybase, auxiliary RPC, query, and tx child commands
	rootCmd.AddCommand(
		sdkserver.StatusCommand(),
		queryCommand(),
		txCommand(),
		// heliosclient.KeyCommands(app.DefaultNodeHome),
		chainclient.KeyCommands(app.DefaultNodeHome),
	)
	rootCmd, err := srvflags.AddTxFlags(rootCmd)
	if err != nil {
		panic(err)
	}

	// add rosetta
	rootCmd.AddCommand(rosettaCmd.RosettaCommand(encodingConfig.InterfaceRegistry, encodingConfig.Codec))

	autoCliOpts := tempApp.AutoCliOpts()
	initClientCtx, _ = clientcfg.ReadFromClientConfig(initClientCtx)
	autoCliOpts.ClientCtx = initClientCtx

	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	return rootCmd, encodingConfig
}

func addModuleInitFlags(startCmd *cobra.Command) {
	crisis.AddModuleInitFlags(startCmd)
	wasm.AddModuleInitFlags(startCmd)
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.QueryEventForTxCmd(),
		rpc.ValidatorCommand(),
		authcmd.QueryTxsByEventsCmd(),
		sdkserver.QueryBlockCmd(),
		authcmd.QueryTxCmd(),
		sdkserver.QueryBlockResultsCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
		authcmd.GetSimulateCmd(),
	)

	// DefaultGasAdjustment value to use as default in gas-adjustment flag
	//flags.DefaultGasAdjustment = servercfg.DefaultGasAdjustment TODO: CHECK IF NECESSARY TO ADD

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	baseDenom, err := sdk.GetBaseDenom()
	if err != nil {
		// NOTE: We need to provide a default base denom for the tempApp created by the RootCmd
		// FIXME: if we remove the min gas price default config, this will no longer be needed
		baseDenom = heliostypes.BaseDenom
	}
	customAppTemplate, customAppConfig := servercfg.AppConfig(baseDenom)

	srvCfg, ok := customAppConfig.(servercfg.Config)
	if !ok {
		panic(fmt.Errorf("unknown app config type %T", customAppConfig))
	}

	srvCfg.StateSync.SnapshotInterval = 5000
	srvCfg.StateSync.SnapshotKeepRecent = 2
	srvCfg.IAVLDisableFastNode = false

	return customAppTemplate, srvCfg
}

type appCreator struct {
	encCfg sdktestutil.TestEncodingConfig
}

// newApp is an appCreator
func (a appCreator) newApp(logger log.Logger, db dbm.DB, traceStore io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
	var cache storetypes.MultiStorePersistentCache

	if cast.ToBool(appOpts.Get(sdkserver.FlagInterBlockCache)) {
		cache = store.NewCommitKVStoreCacheManager()
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(sdkserver.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	pruningOpts, err := sdkserver.GetPruningOptionsFromFlags(appOpts)
	if err != nil {
		panic(err)
	}

	home := cast.ToString(appOpts.Get(flags.FlagHome))
	snapshotDir := filepath.Join(home, "data", "snapshots")
	snapshotDB, err := dbm.NewDB("metadata", sdkserver.GetAppDBBackend(appOpts), snapshotDir)
	if err != nil {
		panic(err)
	}

	options := make([]func(*baseapp.BaseApp), 0)

	if cast.ToBool(appOpts.Get(sdkserver.FlagDumpCommitDebugExecutionTrace)) {
		dataDir := filepath.Join(home, "data")
		traceDB, err := dbm.NewDB("trace", sdkserver.GetAppDBBackend(appOpts), dataDir)
		if err != nil {
			panic(err)
		}

		options = append(options, baseapp.SetTraceDB(traceDB))
		options = append(options, baseapp.SetDumpCommitDebugExecutionTrace(true))
	}

	snapshotStore, err := snapshots.NewStore(snapshotDB, snapshotDir)
	if err != nil {
		panic(err)
	}

	snapshotOptions := snapshottypes.NewSnapshotOptions(
		cast.ToUint64(appOpts.Get(sdkserver.FlagStateSyncSnapshotInterval)),
		cast.ToUint32(appOpts.Get(sdkserver.FlagStateSyncSnapshotKeepRecent)),
	)

	// Setup chainId
	chainID := cast.ToString(appOpts.Get(flags.FlagChainID))
	if len(chainID) == 0 {
		v := viper.New()
		v.AddConfigPath(filepath.Join(home, "config"))
		v.SetConfigName("client")
		v.SetConfigType("toml")
		if err := v.ReadInConfig(); err != nil {
			panic(err)
		}
		conf := new(clientcfg.ClientConfig)
		if err := v.Unmarshal(conf); err != nil {
			panic(err)
		}
		chainID = conf.ChainID
	}

	options = append(options, baseapp.SetRootDir(home))
	options = append(options, baseapp.SetPruning(pruningOpts))

	options = append(options, baseapp.SetMinGasPrices(cast.ToString(appOpts.Get(sdkserver.FlagMinGasPrices))))
	options = append(options, baseapp.SetHaltHeight(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltHeight))))
	options = append(options, baseapp.SetHaltTime(cast.ToUint64(appOpts.Get(sdkserver.FlagHaltTime))))
	options = append(options, baseapp.SetMinRetainBlocks(cast.ToUint64(appOpts.Get(sdkserver.FlagMinRetainBlocks))))
	options = append(options, baseapp.SetInterBlockCache(cache))
	options = append(options, baseapp.SetTrace(cast.ToBool(appOpts.Get(sdkserver.FlagTrace))))
	options = append(options, baseapp.SetIndexEvents(cast.ToStringSlice(appOpts.Get(sdkserver.FlagIndexEvents))))
	options = append(options, baseapp.SetSnapshot(snapshotStore, snapshotOptions))
	options = append(options, baseapp.SetIAVLCacheSize(cast.ToInt(appOpts.Get(sdkserver.FlagIAVLCacheSize))))
	options = append(options, baseapp.SetIAVLDisableFastNode(cast.ToBool(appOpts.Get(sdkserver.FlagDisableIAVLFastNode))))
	options = append(options, baseapp.SetChainID(chainID))

	if cast.ToBool(appOpts.Get(sdkserver.FlagBackupEnable)) {
		options = append(options, baseapp.SetBackupConfig(
			cast.ToBool(appOpts.Get(sdkserver.FlagBackupEnable)),
			cast.ToUint64(appOpts.Get(sdkserver.FlagBackupBlockInterval)),
			cast.ToString(appOpts.Get(sdkserver.FlagBackupDir)),
			cast.ToUint64(appOpts.Get(sdkserver.FlagBackupMinRetainBackups)),
		))
	}

	heliosApp := app.NewHeliosApp(
		logger, db, traceStore, true,
		appOpts,
		options...,
	)

	return heliosApp
}

// appExport creates a new simapp (optionally at a given height)
// and exports state.
func (a appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	var heliosApp *app.HeliosApp
	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home not set")
	}

	if height != -1 {
		heliosApp = app.NewHeliosApp(logger, db, traceStore, false, appOpts)

		if err := heliosApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	} else {
		heliosApp = app.NewHeliosApp(logger, db, traceStore, true, appOpts)
	}

	return heliosApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

// initTendermintConfig helps to override default Tendermint Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()
	cfg.Consensus.TimeoutCommit = time.Second * 3

	// to put a higher strain on node memory, use these values:
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

func tempDir(defaultHome string) string {
	dir, err := os.MkdirTemp("", "helios")
	if err != nil {
		dir = defaultHome
	}
	defer os.RemoveAll(dir)

	return dir
}
