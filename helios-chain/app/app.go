package app

import (
	"io"

	// "io/fs"
	// "net/http"
	"os"
	"path/filepath"
	"strconv"

	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/migrations/legacytx"
	tx "github.com/cosmos/cosmos-sdk/x/auth/tx/config"

	// "github.com/gorilla/mux"

	"github.com/spf13/cast"

	abci "github.com/cometbft/cometbft/abci/types"
	tmos "github.com/cometbft/cometbft/libs/os"
	"github.com/cometbft/cometbft/libs/pubsub"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	reflectionv1 "cosmossdk.io/api/cosmos/reflection/v1"
	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	testdata_pulsar "github.com/cosmos/cosmos-sdk/testutil/testdata/testpb"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	runtimeservices "github.com/cosmos/cosmos-sdk/runtime/services"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsims "github.com/cosmos/cosmos-sdk/x/auth/simulation"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	authzcdc "github.com/cosmos/cosmos-sdk/x/authz/codec"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	authzmodule "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensusparamkeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward"
	packetforwardkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/keeper"
	packetforwardtypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8/packetforward/types"
	"github.com/cosmos/ibc-go/modules/capability"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcclient "github.com/cosmos/ibc-go/v8/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"

	"github.com/Helios-Chain-Labs/metrics"

	// "helios-core/client/docs" // removed
	"helios-core/helios-chain/app/ante"
	"helios-core/helios-chain/stream"
	chaintypes "helios-core/helios-chain/types"
	hyperion "helios-core/helios-chain/x/hyperion"
	hyperionKeeper "helios-core/helios-chain/x/hyperion/keeper"
	hyperiontypes "helios-core/helios-chain/x/hyperion/types"
	"helios-core/helios-chain/x/tokenfactory"
	tokenfactorykeeper "helios-core/helios-chain/x/tokenfactory/keeper"
	tokenfactorytypes "helios-core/helios-chain/x/tokenfactory/types"

	epochskeeper "helios-core/helios-chain/x/epochs/keeper"
	erc20keeper "helios-core/helios-chain/x/erc20/keeper"
	erc20types "helios-core/helios-chain/x/erc20/types"

	chronoskeeper "helios-core/helios-chain/x/chronos/keeper"
	chronostypes "helios-core/helios-chain/x/chronos/types"

	"helios-core/helios-chain/x/evm"
	evmkeeper "helios-core/helios-chain/x/evm/keeper"
	evmtypes "helios-core/helios-chain/x/evm/types"

	epochstypes "helios-core/helios-chain/x/epochs/types"
	"helios-core/helios-chain/x/feemarket"
	feemarketkeeper "helios-core/helios-chain/x/feemarket/keeper"
	feemarkettypes "helios-core/helios-chain/x/feemarket/types"

	inflationkeeper "helios-core/helios-chain/x/inflation/v1/keeper"
	inflationtypes "helios-core/helios-chain/x/inflation/v1/types"

	srvflags "helios-core/helios-chain/server/flags"

	chronos "helios-core/helios-chain/x/chronos"
	epochs "helios-core/helios-chain/x/epochs"
	erc20 "helios-core/helios-chain/x/erc20"

	post "helios-core/helios-chain/app/post"

	transferkeeper "helios-core/helios-chain/x/ibc/transfer/keeper"

	//stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	// NOTE: override ICS20 keeper to support IBC transfers of ERC20 tokens
	transfer "helios-core/helios-chain/x/ibc/transfer"

	stakingkeeper "helios-core/helios-chain/x/staking/keeper"

	ratelimit "github.com/cosmos/ibc-apps/modules/rate-limiting/v8"
	ratelimitkeeper "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/keeper"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v8/types"

	staking "helios-core/helios-chain/x/staking"

	encoding "helios-core/helios-chain/encoding"

	evmostypes "helios-core/helios-chain/types"

	ethante "helios-core/helios-chain/app/ante/evm"

	sdkstaking "github.com/cosmos/cosmos-sdk/x/staking"
)

func init() {
	// set the address prefixes
	sdkConfig := sdk.GetConfig()
	chaintypes.SetBech32Prefixes(sdkConfig)
	chaintypes.SetBip44CoinType(sdkConfig)

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	customHomeDir := os.Getenv("HELIADES_HOME")
	if customHomeDir != "" {
		DefaultNodeHome = filepath.Join(userHomeDir, customHomeDir)
	} else {
		DefaultNodeHome = filepath.Join(userHomeDir, ".heliades")
	}
}

const appName = "helios-chain"

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		genutil.AppModuleBasic{GenTxValidator: genutiltypes.DefaultMessageValidator},
		bank.AppModuleBasic{},
		capability.AppModuleBasic{},
		staking.AppModuleBasic{AppModuleBasic: &sdkstaking.AppModuleBasic{}},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic([]govclient.ProposalHandler{paramsclient.ProposalHandler}),
		consensus.AppModuleBasic{},
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctm.AppModuleBasic{},
		ica.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		vesting.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		authzmodule.AppModuleBasic{},
		packetforward.AppModuleBasic{},
		hyperion.AppModuleBasic{},
		tokenfactory.AppModuleBasic{},
		erc20.AppModuleBasic{},
		chronos.AppModuleBasic{},
	)

	// module account permissions
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     {authtypes.Burner},
		distrtypes.ModuleName:          nil,
		icatypes.ModuleName:            nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BoostedPoolName:   {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:         nil,
		hyperiontypes.ModuleName:       {authtypes.Minter, authtypes.Burner},
		tokenfactorytypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
		erc20types.ModuleName:          {authtypes.Minter, authtypes.Burner},
		evmtypes.ModuleName:            {authtypes.Minter, authtypes.Burner}, // used for secure addition and subtraction of balance using module account
		ratelimittypes.ModuleName:      nil,
		// feemarkettypes.ModuleName:      nil,
	}

	// module accounts that are allowed to receive tokens
	allowedReceivingModAcc = map[string]bool{
		distrtypes.ModuleName:        true,
		hyperiontypes.ModuleName:     true,
		tokenfactorytypes.ModuleName: true,
	}
)

var _ runtime.AppI = (*HeliosApp)(nil)

// HeliosApp implements an extended ABCI application.
type HeliosApp struct {
	*baseapp.BaseApp
	amino             *codec.LegacyAmino
	codec             codec.Codec
	interfaceRegistry types.InterfaceRegistry
	txConfig          client.TxConfig

	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tKeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// cosmos keepers
	AuthzKeeper           authzkeeper.Keeper
	AccountKeeper         authkeeper.AccountKeeper
	BankKeeper            bankkeeper.Keeper
	CapabilityKeeper      *capabilitykeeper.Keeper
	StakingKeeper         *stakingkeeper.Keeper
	SlashingKeeper        slashingkeeper.Keeper
	MintKeeper            mintkeeper.Keeper
	DistrKeeper           distrkeeper.Keeper
	GovKeeper             govkeeper.Keeper
	CrisisKeeper          *crisiskeeper.Keeper
	UpgradeKeeper         *upgradekeeper.Keeper
	ParamsKeeper          paramskeeper.Keeper
	EvidenceKeeper        evidencekeeper.Keeper
	ConsensusParamsKeeper consensusparamkeeper.Keeper

	// helios keepers
	TokenFactoryKeeper tokenfactorykeeper.Keeper
	HyperionKeeper     hyperionKeeper.Keeper

	// ibc keepers
	IBCKeeper           *ibckeeper.Keeper // IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCFeeKeeper        ibcfeekeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	TransferKeeper      transferkeeper.Keeper
	FeeGrantKeeper      feegrantkeeper.Keeper
	PacketForwardKeeper *packetforwardkeeper.Keeper

	// scoped keepers
	ScopedIBCKeeper      capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper  capabilitykeeper.ScopedKeeper

	BasicModuleManager module.BasicManager
	mm                 *module.Manager
	sm                 *module.SimulationManager

	configurator module.Configurator

	// stream server
	ChainStreamServer *stream.StreamServer
	EventPublisher    *stream.Publisher

	// ethermint keepers
	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	InflationKeeper inflationkeeper.Keeper

	// Helios keepers
	Erc20Keeper   erc20keeper.Keeper
	EpochsKeeper  epochskeeper.Keeper
	ChronosKeeper chronoskeeper.Keeper

	RateLimitKeeper ratelimitkeeper.Keeper
}

// NewHeliosApp returns a reference to a new initialized Helios application.
func NewHeliosApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *HeliosApp {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	app := initHeliosApp(appName, logger, db, traceStore, baseAppOptions...)

	app.initKeepers(authority, appOpts)
	app.initManagers()
	app.registerUpgradeHandlers()

	app.configurator = module.NewConfigurator(app.codec, app.MsgServiceRouter(), app.GRPCQueryRouter())

	if err := app.mm.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

	autocliv1.RegisterQueryServer(app.GRPCQueryRouter(), runtimeservices.NewAutoCLIQueryService(app.mm.Modules))

	reflectionSvc, err := runtimeservices.NewReflectionService()
	if err != nil {
		panic(err)
	}

	reflectionv1.RegisterReflectionServiceServer(app.GRPCQueryRouter(), reflectionSvc)

	testdata_pulsar.RegisterQueryServer(app.GRPCQueryRouter(), testdata_pulsar.QueryImpl{})

	// initialize stores
	app.MountKVStores(app.keys)
	app.MountTransientStores(app.tKeys)
	app.MountMemoryStores(app.memKeys)

	// load state streaming if enabled
	if err := app.RegisterStreamingServices(appOpts, app.keys); err != nil {
		panic("failed to load state streaming: " + err.Error())
	}

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// use Helios's custom AnteHandler
	skipAnteHandlers := cast.ToBool(appOpts.Get("SkipAnteHandlers"))
	if !skipAnteHandlers {
		maxGasWanted := cast.ToUint64(appOpts.Get(srvflags.EVMMaxTxGasWanted))

		options := ante.HandlerOptions{
			Cdc:                    app.codec,
			AccountKeeper:          app.AccountKeeper,
			BankKeeper:             app.BankKeeper,
			ExtensionOptionChecker: evmostypes.HasDynamicFeeExtensionOption,
			EvmKeeper:              app.EvmKeeper,
			StakingKeeper:          app.StakingKeeper,
			FeegrantKeeper:         app.FeeGrantKeeper,
			DistributionKeeper:     app.DistrKeeper,
			IBCKeeper:              app.IBCKeeper,
			FeeMarketKeeper:        app.FeeMarketKeeper,
			SignModeHandler:        app.txConfig.SignModeHandler(),
			SigGasConsumer:         ante.SigVerificationGasConsumer,
			MaxTxGasWanted:         maxGasWanted,
			TxFeeChecker:           ethante.NewDynamicFeeChecker(app.FeeMarketKeeper),
		}

		if err := options.Validate(); err != nil {
			panic(err)
		}
		app.SetAnteHandler(ante.NewAnteHandler(options))
		app.setPostHandler()
		// app.setupUpgradeHandlers()
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(err.Error())
		}
	}
	// Applications that wish to enforce statically created ScopedKeepers should call `Seal` after creating
	// their scoped modules in `NewApp` with `ScopeToModule`
	app.CapabilityKeeper.Seal()

	bus := pubsub.NewServer()
	app.EventPublisher = stream.NewPublisher(app.StreamEvents, bus)
	app.ChainStreamServer = stream.NewChainStreamServer(bus, appOpts)

	authzcdc.GlobalCdc = codec.NewProtoCodec(app.interfaceRegistry)
	ante.GlobalCdc = codec.NewProtoCodec(app.interfaceRegistry)
	legacytx.RegressionTestingAminoCodec = app.amino

	return app
}

func initHeliosApp(
	name string,
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	baseAppOptions ...func(*baseapp.BaseApp),
) *HeliosApp {
	var (
		// encodingConfig    = helioscodectypes.MakeEncodingConfig()
		encodingConfig    = encoding.MakeConfig()
		appCodec          = encodingConfig.Codec
		legacyAmino       = encodingConfig.Amino
		interfaceRegistry = encodingConfig.InterfaceRegistry

		keys = storetypes.NewKVStoreKeys(
			// SDK keys
			authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
			minttypes.StoreKey, distrtypes.StoreKey, slashingtypes.StoreKey,
			govtypes.StoreKey, paramstypes.StoreKey, ibcexported.StoreKey,
			upgradetypes.StoreKey, evidencetypes.StoreKey, ibctransfertypes.StoreKey,
			capabilitytypes.StoreKey, feegrant.StoreKey, authzkeeper.StoreKey,
			icahosttypes.StoreKey, ibcfeetypes.StoreKey, crisistypes.StoreKey,
			consensustypes.StoreKey, packetforwardtypes.StoreKey,
			// Helios keys
			hyperiontypes.StoreKey,
			tokenfactorytypes.StoreKey,
			epochstypes.StoreKey,
			inflationtypes.StoreKey,
			// Add missing EVM-related keys
			evmtypes.StoreKey,
			feemarkettypes.StoreKey,
			erc20types.StoreKey,
			ratelimittypes.StoreKey,
			chronostypes.StoreKey,
		)

		tKeys = storetypes.NewTransientStoreKeys(
			paramstypes.TStoreKey,
			banktypes.TStoreKey,
			// Add missing EVM-related transient keys
			evmtypes.TransientKey,
			feemarkettypes.TransientKey,
		)

		memKeys = storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)
	)

	bApp := baseapp.NewBaseApp(
		name,
		logger,
		db,
		encodingConfig.TxConfig.TxDecoder(), // NOTE we use custom Helios transaction decoder that supports the sdk.Tx interface instead of sdk.StdTx
		baseAppOptions...,
	)

	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetName(version.Name)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	if err := InitializeAppConfiguration(bApp.ChainID()); err != nil {
		panic(err)
	}

	app := &HeliosApp{
		BaseApp:           bApp,
		amino:             legacyAmino,
		codec:             appCodec,
		interfaceRegistry: interfaceRegistry,
		txConfig:          encodingConfig.TxConfig,
		keys:              keys,
		tKeys:             tKeys,
		memKeys:           memKeys,
	}

	return app
}

func (app *HeliosApp) GetBaseApp() *baseapp.BaseApp { return app.BaseApp }

func (app *HeliosApp) GetIBCKeeper() *ibckeeper.Keeper { return app.IBCKeeper }

func (app *HeliosApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return app.ScopedIBCKeeper
}

func (app *HeliosApp) GetTxConfig() client.TxConfig { return app.txConfig }

// AutoCliOpts returns the autocli options for the app.
func (app *HeliosApp) AutoCliOpts() autocli.AppOptions {
	modules := make(map[string]appmodule.AppModule, 0)
	for _, m := range app.mm.Modules {
		if moduleWithName, ok := m.(module.HasName); ok {
			moduleName := moduleWithName.Name()
			if appModule, ok := moduleWithName.(appmodule.AppModule); ok {
				modules[moduleName] = appModule
			}
		}
	}

	return autocli.AppOptions{
		Modules:               modules,
		ModuleOptions:         runtimeservices.ExtractAutoCLIOptions(app.mm.Modules),
		AddressCodec:          authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddressCodec: authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

// Name returns the name of the App
func (app *HeliosApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker updates every begin block
func (app *HeliosApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, metrics.Tags{"svc": "app", "height": strconv.Itoa(int(ctx.BlockHeight()))})
	defer doneFn()
	return app.mm.BeginBlock(ctx)
}

// PreBlocker application updates every pre block
func (app *HeliosApp) PreBlocker(ctx sdk.Context, _ *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	return app.mm.PreBlock(ctx)
}

// EndBlocker updates every end block
func (app *HeliosApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	ctx, doneFn := metrics.ReportFuncCallAndTimingSdkCtx(ctx, metrics.Tags{"svc": "app", "height": strconv.Itoa(int(ctx.BlockHeight()))})
	defer doneFn()
	return app.mm.EndBlock(ctx)
}

// InitChainer updates at chain initialization
func (app *HeliosApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState GenesisState
	app.amino.MustUnmarshalJSON(req.AppStateBytes, &genesisState)
	if err := app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap()); err != nil {
		return nil, err
	}

	return app.mm.InitGenesis(ctx, app.codec, genesisState)
}

func (app *HeliosApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// LoadHeight loads state at a particular height
func (app *HeliosApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *HeliosApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedAddrs returns all the app's module account addresses that are not
// allowed to receive external tokens.
func (app *HeliosApp) BlockedAddrs() map[string]bool {
	blockedAddrs := make(map[string]bool)
	for acc := range maccPerms {
		blockedAddrs[authtypes.NewModuleAddress(acc).String()] = !allowedReceivingModAcc[acc]
	}

	return blockedAddrs
}

// LegacyAmino returns HeliosApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *HeliosApp) LegacyAmino() *codec.LegacyAmino {
	return app.amino
}

// AppCodec returns HeliosApp's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *HeliosApp) AppCodec() codec.Codec {
	return app.codec
}

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *HeliosApp) DefaultGenesis() evmostypes.GenesisState {
	return app.BasicModuleManager.DefaultGenesis(app.codec)
}

// InterfaceRegistry returns HeliosApp's InterfaceRegistry
func (app *HeliosApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// TxConfig returns SimApp's TxConfig
func (app *HeliosApp) TxConfig() client.TxConfig {
	return app.txConfig
}

// GetKey returns the KVStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeliosApp) GetKey(storeKey string) *storetypes.KVStoreKey {
	return app.keys[storeKey]
}

func (app *HeliosApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.StakingKeeper
}

// GetTKey returns the TransientStoreKey for the provided store key.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeliosApp) GetTKey(storeKey string) *storetypes.TransientStoreKey {
	return app.tKeys[storeKey]
}

// GetMemKey returns the MemStoreKey for the provided mem key.
//
// NOTE: This is solely used for testing purposes.
func (app *HeliosApp) GetMemKey(storeKey string) *storetypes.MemoryStoreKey {
	return app.memKeys[storeKey]
}

// GetSubspace returns a param subspace for a given module name.
//
// NOTE: This is solely to be used for testing purposes.
func (app *HeliosApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// SimulationManager implements the SimulationApp interface
func (app *HeliosApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *HeliosApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx

	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	cmtservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register node gRPC service for grpc-gateway.
	nodeservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register grpc-gateway routes for all modules.
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register swagger API from root so that other applications can override easily
	// if err := RegisterSwaggerAPI(clientCtx, apiSvr.Router, apiConfig.Swagger); err != nil {
	// 	panic(err)
	// }
}

// // RegisterSwaggerAPI provides a common function which registers swagger route with API Server
// func RegisterSwaggerAPI(_ client.Context, rtr *mux.Router, swaggerEnabled bool) error {
// 	if !swaggerEnabled {
// 		return nil
// 	}

// 	root, err := fs.Sub(docs.SwaggerUI, "swagger-ui")
// 	if err != nil {
// 		return err
// 	}

// 	staticServer := http.FileServer(http.FS(root))
// 	rtr.PathPrefix("/swagger/").Handler(http.StripPrefix("/swagger/", staticServer))
// 	return nil
// }

func (app *HeliosApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

func (app *HeliosApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(clientCtx, app.BaseApp.GRPCQueryRouter(), app.interfaceRegistry, app.Query)
}

func (app *HeliosApp) initKeepers(authority string, appOpts servertypes.AppOptions) {
	app.ParamsKeeper = initParamsKeeper(
		app.codec,
		app.amino,
		app.keys[paramstypes.StoreKey],
		app.tKeys[paramstypes.TStoreKey],
	)

	skipUpgradeHeights := map[int64]bool{}
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		runtime.NewKVStoreService(app.keys[upgradetypes.StoreKey]),
		app.codec,
		cast.ToString(appOpts.Get(flags.FlagHome)),
		app.BaseApp,
		authority,
	)

	app.ConsensusParamsKeeper = consensusparamkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[consensustypes.StoreKey]),
		authority,
		runtime.EventService{},
	)

	app.SetParamStore(app.ConsensusParamsKeeper.ParamsStore)

	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		app.codec,
		app.keys[capabilitytypes.StoreKey],
		app.memKeys[capabilitytypes.MemStoreKey],
	)

	app.AccountKeeper = authkeeper.NewAccountKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[authtypes.StoreKey]),
		chaintypes.ProtoAccount, // use custom Ethermint account
		maccPerms,
		authcodec.NewBech32Codec(chaintypes.Bech32Prefix),
		chaintypes.Bech32Prefix,
		authority,
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[banktypes.StoreKey]),
		runtime.NewTransientKVStoreService(app.tKeys[banktypes.TStoreKey]),
		app.AccountKeeper,
		app.BlockedAddrs(),
		authority,
		app.Logger(),
	)

	// SDK v0.50
	// Legacy app wiring: to enable SignMode_SIGN_MODE_TEXTUAL app tx config must be updated after bank keeper init
	txConfigOpts := authtx.ConfigOptions{
		EnabledSignModes:           append(authtx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL),
		TextualCoinMetadataQueryFn: tx.NewBankKeeperCoinMetadataQueryFn(app.BankKeeper),
	}

	txConfig, err := authtx.NewTxConfigWithOptions(app.codec, txConfigOpts)
	if err != nil {
		panic("failed to update app tx config: " + err.Error())
	}

	app.txConfig = txConfig

	app.AuthzKeeper = authzkeeper.NewKeeper(
		runtime.NewKVStoreService(app.keys[authzkeeper.StoreKey]),
		app.codec,
		app.MsgServiceRouter(),
		app.AccountKeeper,
	)

	app.AuthzKeeper = app.AuthzKeeper.SetBankKeeper(app.BankKeeper)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.Erc20Keeper,
		authority,
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		authcodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	)

	app.MintKeeper = mintkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authority,
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authority,
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		app.codec,
		app.amino,
		runtime.NewKVStoreService(app.keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authority,
	)

	app.CrisisKeeper = crisiskeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[crisistypes.StoreKey]),
		cast.ToUint(appOpts.Get(server.FlagInvCheckPeriod)),
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authority,
		app.AccountKeeper.AddressCodec(),
	)

	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	app.GovKeeper = *govkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		app.MsgServiceRouter(),
		govtypes.DefaultConfig(),
		authority,
	)

	app.ScopedIBCKeeper = app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.IBCKeeper = ibckeeper.NewKeeper(
		app.codec,
		app.keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
		authority,
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[feegrant.StoreKey]),
		&app.AccountKeeper,
	)

	app.HyperionKeeper = hyperionKeeper.NewKeeper(
		app.codec,
		app.keys[hyperiontypes.StoreKey],
		app.StakingKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.DistrKeeper,
		authority,
		app.AccountKeeper,
		app.Erc20Keeper,
	)

	app.TokenFactoryKeeper = tokenfactorykeeper.NewKeeper(
		app.keys[tokenfactorytypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper.(bankkeeper.BaseKeeper).WithMintCoinsRestriction(tokenfactorytypes.NewTokenFactoryDenomMintCoinsRestriction()),
		app.DistrKeeper,
		authority,
	)

	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		app.codec,
		app.keys[ibcfeetypes.StoreKey],
		app.IBCKeeper.ChannelKeeper, // may be replaced with IBC middleware
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	// Initialize packet forward middleware router
	app.PacketForwardKeeper = packetforwardkeeper.NewKeeper(
		app.codec,
		app.keys[packetforwardtypes.StoreKey],
		app.TransferKeeper, // Will be zero-value here. Reference is set later on with SetTransferKeeper.
		app.IBCKeeper.ChannelKeeper,
		app.DistrKeeper,
		app.BankKeeper,
		app.IBCFeeKeeper,
		authority,
	)

	// Create the rate limit keeper
	app.RateLimitKeeper = *ratelimitkeeper.NewKeeper(
		app.codec,
		runtime.NewKVStoreService(app.keys[ratelimittypes.StoreKey]),
		app.GetSubspace(ratelimittypes.ModuleName),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.BankKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper, // ICS4Wrapper
	)

	// Create Transfer Keepers
	app.ScopedTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)

	app.TransferKeeper = transferkeeper.NewKeeper(
		app.codec, app.keys[ibctransfertypes.StoreKey], app.GetSubspace(ibctransfertypes.ModuleName),
		app.RateLimitKeeper, // ICS4 Wrapper: ratelimit IBC middleware
		app.IBCKeeper.ChannelKeeper, app.IBCKeeper.PortKeeper,
		app.AccountKeeper, app.BankKeeper, app.ScopedTransferKeeper,
		app.Erc20Keeper, // Add ERC20 Keeper for ERC20 transfers
		authority,
	)

	app.PacketForwardKeeper.SetTransferKeeper(app.TransferKeeper)

	app.ScopedICAHostKeeper = app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		app.codec,
		app.keys[icahosttypes.StoreKey],
		app.GetSubspace(icahosttypes.SubModuleName),
		app.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
		authority,
	)

	app.ICAHostKeeper.WithQueryRouter(app.GRPCQueryRouter())

	// Create Transfer Stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(app.TransferKeeper)
	transferStack = ratelimit.NewIBCMiddleware(app.RateLimitKeeper, transferStack)
	transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack)
	transferStack = packetforward.NewIBCMiddleware(transferStack,
		app.PacketForwardKeeper,
		0,
		packetforwardkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		packetforwardkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
	)
	transferStack = ibcfee.NewIBCMiddleware(transferStack, app.IBCFeeKeeper)

	// Create Interchain Accounts Stack
	// SendPacket, since it is originating from the application to core IBC:
	// icaAuthModuleKeeper.SendTx -> icaController.SendPacket -> fee.SendPacket -> channel.SendPacket

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	var icaHostStack porttypes.IBCModule
	icaHostStack = icahost.NewIBCModule(app.ICAHostKeeper)
	icaHostStack = ibcfee.NewIBCMiddleware(icaHostStack, app.IBCFeeKeeper)

	// Create static IBC router, add ibctransfer route, then set and seal it
	ibcRouter := porttypes.NewRouter().
		AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack)

	// Setting Router will finalize all routes by sealing router
	// No more routes can be added
	app.IBCKeeper.SetRouter(ibcRouter)

	// Create FeeMarket keeper

	// ALL EVM

	// Create Ethermint keepers
	app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		app.codec, authtypes.NewModuleAddress(govtypes.ModuleName),
		app.keys[feemarkettypes.StoreKey],
		app.tKeys[feemarkettypes.TransientKey],
		app.GetSubspace(feemarkettypes.ModuleName),
	)

	app.InflationKeeper = *inflationkeeper.NewKeeper(app.codec, app.keys[inflationtypes.StoreKey])

	tracer := cast.ToString(appOpts.Get(srvflags.EVMTracer))

	evmKeeper := evmkeeper.NewKeeper(
		app.codec, app.keys[evmtypes.StoreKey], app.tKeys[evmtypes.TransientKey], authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.FeeMarketKeeper,
		// FIX: Temporary solution to solve keeper interdependency while new precompile module
		// is being developed.
		app.Erc20Keeper,
		tracer, app.GetSubspace(evmtypes.ModuleName),
	)
	app.EvmKeeper = evmKeeper

	app.ChronosKeeper = *chronoskeeper.NewKeeper(
		app.codec,
		app.keys[chronostypes.StoreKey],
		app.keys[chronostypes.MemStoreKey],
		app.AccountKeeper,
		app.EvmKeeper,
		app.BankKeeper,
	)

	erc20Keeper := erc20keeper.NewKeeper(
		app.keys[erc20types.StoreKey], app.codec, authtypes.NewModuleAddress(govtypes.ModuleName),
		app.AccountKeeper, app.BankKeeper, app.EvmKeeper, app.StakingKeeper,
		app.AuthzKeeper, &app.TransferKeeper,
	)
	app.Erc20Keeper = erc20Keeper

	app.StakingKeeper.SetErc20Keeper(app.Erc20Keeper)
	app.EvmKeeper.SetErc20Keeper(app.Erc20Keeper)
	// app.HyperionKeeper.SetErc20Keeper(app.Erc20Keeper)

	app.HyperionKeeper = hyperionKeeper.NewKeeper(
		app.codec,
		app.keys[hyperiontypes.StoreKey],
		app.StakingKeeper,
		app.BankKeeper,
		app.SlashingKeeper,
		app.DistrKeeper,
		authority,
		app.AccountKeeper,
		app.Erc20Keeper,
	)

	epochsKeeper := epochskeeper.NewKeeper(app.codec, app.keys[epochstypes.StoreKey])

	app.StakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(
		app.DistrKeeper.Hooks(),
		app.SlashingKeeper.Hooks(),
		app.HyperionKeeper.Hooks(),
	))

	app.EpochsKeeper = *epochsKeeper.SetHooks(
		epochskeeper.NewMultiEpochHooks(
		// insert epoch hooks receivers here
		//app.InflationKeeper.Hooks(),
		),
	)

	evmKeeper.WithStaticPrecompiles(
		evmkeeper.NewAvailableStaticPrecompiles(
			*app.StakingKeeper,
			app.DistrKeeper,
			app.BankKeeper,
			app.Erc20Keeper,
			app.AuthzKeeper,
			app.TransferKeeper,
			app.IBCKeeper.ChannelKeeper,
			app.GovKeeper,
			app.ChronosKeeper,
			app.HyperionKeeper,
		),
	)

	// register the proposal types
	govRouter := govv1beta1.NewRouter().
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(app.IBCKeeper.ClientKeeper)). //nolint:staticcheck // SA1019 Existing use of deprecated but supported function
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(app.Erc20Keeper)).
		AddRoute(minttypes.RouterKey, mint.NewProposalHandler(app.MintKeeper))

	app.GovKeeper.SetLegacyRouter(govRouter)
}

func (app *HeliosApp) setPostHandler() {
	options := post.HandlerOptions{
		FeeCollectorName: authtypes.FeeCollectorName,
		BankKeeper:       app.BankKeeper,
	}

	if err := options.Validate(); err != nil {
		panic(err)
	}

	app.SetPostHandler(post.NewPostHandler(options))
}

func (app *HeliosApp) initManagers() {
	// NOTE: we may consider parsing `appOpts` inside module constructors. For the moment
	// we prefer to be more strict in what arguments the modules expect.
	// var skipGenesisInvariants = cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))
	skipGenesisInvariants := true

	transferModule := transfer.NewAppModule(app.TransferKeeper)
	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(

		// SDK app modules
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(app.codec, app.AccountKeeper, authsims.RandomGenesisAccounts, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(app.codec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		capability.NewAppModule(app.codec, *app.CapabilityKeeper, false),
		crisis.NewAppModule(app.CrisisKeeper, skipGenesisInvariants, app.GetSubspace(crisistypes.ModuleName)), // always be last to make sure that it checks for all invariants and not only part of them
		feegrantmodule.NewAppModule(app.codec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(app.codec, &app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(app.codec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(app.codec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(app.codec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(app.codec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		authzmodule.NewAppModule(app.codec, app.AuthzKeeper, app.AccountKeeper, app.BankKeeper, app.interfaceRegistry),
		consensus.NewAppModule(app.codec, app.ConsensusParamsKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		transferModule,
		ratelimit.NewAppModule(app.codec, app.RateLimitKeeper),
		// Ethermint app modules
		evm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.GetSubspace(evmtypes.ModuleName)),
		feemarket.NewAppModule(app.FeeMarketKeeper, app.GetSubspace(feemarkettypes.ModuleName)),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ibctm.NewAppModule(),
		ica.NewAppModule(nil, &app.ICAHostKeeper),
		packetforward.NewAppModule(app.PacketForwardKeeper, app.GetSubspace(packetforwardtypes.ModuleName)),
		// Helios app modules
		hyperion.NewAppModule(app.HyperionKeeper, app.BankKeeper, app.GetSubspace(hyperiontypes.ModuleName)),
		tokenfactory.NewAppModule(app.TokenFactoryKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(tokenfactorytypes.ModuleName)),
		erc20.NewAppModule(app.Erc20Keeper, app.AccountKeeper,
			app.GetSubspace(erc20types.ModuleName)),
		epochs.NewAppModule(app.codec, app.EpochsKeeper),
		chronos.NewAppModule(app.codec, app.ChronosKeeper),
	)

	// BasicModuleManager defines the module BasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration and genesis verification.
	// By default it is composed of all the module from the module manager.
	// Additionally, app module basics can be overwritten by passing them as argument.
	app.BasicModuleManager = module.NewBasicManagerFromManager(app.mm, map[string]module.AppModuleBasic{
		genutiltypes.ModuleName: genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		govtypes.ModuleName:     gov.NewAppModuleBasic([]govclient.ProposalHandler{paramsclient.ProposalHandler}),
	})
	app.BasicModuleManager.RegisterLegacyAminoCodec(app.amino)
	app.BasicModuleManager.RegisterInterfaces(app.interfaceRegistry)

	app.mm.SetOrderPreBlockers(upgradetypes.ModuleName) // NOTE: upgrade module is required to be prioritized
	app.mm.SetOrderBeginBlockers(beginBlockerOrder()...)
	app.mm.SetOrderEndBlockers(endBlockerOrder()...)
	app.mm.SetOrderInitGenesis(initGenesisOrder()...)
	app.mm.RegisterInvariants(app.CrisisKeeper)

	// create the simulation manager and define the order of the modules for deterministic simulations

	// NOTE: this is not required apps that don't use the simulator for fuzz testing
	// transactions
	overrideModules := map[string]module.AppModuleSimulation{
		authtypes.ModuleName: auth.NewAppModule(
			app.codec,
			app.AccountKeeper,
			authsims.RandomGenesisAccounts,
			app.GetSubspace(authtypes.ModuleName),
		),
	}

	app.sm = module.NewSimulationManagerFromAppModules(app.mm.Modules, overrideModules)
	app.sm.RegisterStoreDecoders()
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(
	appCodec codec.BinaryCodec,
	legacyAmino *codec.LegacyAmino,
	key, tkey storetypes.StoreKey,
) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	// SDK subspaces
	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)

	// register the key tables for legacy param subspaces
	keyTable := ibcclienttypes.ParamKeyTable()
	keyTable.RegisterParamSet(&ibcconnectiontypes.Params{})
	paramsKeeper.Subspace(ibcexported.ModuleName).WithKeyTable(keyTable)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName).WithKeyTable(ibctransfertypes.ParamKeyTable())
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName).WithKeyTable(icacontrollertypes.ParamKeyTable())
	paramsKeeper.Subspace(icahosttypes.SubModuleName).WithKeyTable(icahosttypes.ParamKeyTable())

	paramsKeeper.Subspace(packetforwardtypes.ModuleName).WithKeyTable(packetforwardtypes.ParamKeyTable())
	// helios subspaces
	paramsKeeper.Subspace(hyperiontypes.ModuleName)
	paramsKeeper.Subspace(tokenfactorytypes.ModuleName)

	// FIX: do we need a keytable?
	paramsKeeper.Subspace(ratelimittypes.ModuleName)
	// ethermint subspaces
	paramsKeeper.Subspace(evmtypes.ModuleName).WithKeyTable(evmtypes.ParamKeyTable()) //nolint: staticcheck
	paramsKeeper.Subspace(feemarkettypes.ModuleName).WithKeyTable(feemarkettypes.ParamKeyTable())
	// evmos subspaces
	paramsKeeper.Subspace(erc20types.ModuleName)
	paramsKeeper.Subspace(chronostypes.ModuleName)

	return paramsKeeper
}

func initGenesisOrder() []string {
	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	return []string{
		// SDK modules
		capabilitytypes.ModuleName,
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		// Ethermint modules
		evmtypes.ModuleName,
		// NOTE: feemarket module needs to be initialized before genutil module:
		// gentx transactions use MinGasPriceDecorator.AnteHandle
		feemarkettypes.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		paramstypes.ModuleName,
		authz.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		feegrant.ModuleName,
		consensustypes.ModuleName,
		packetforwardtypes.ModuleName,

		// Helios modules
		tokenfactorytypes.ModuleName,
		hyperiontypes.ModuleName,
		erc20types.ModuleName,
		epochstypes.ModuleName,
		ratelimittypes.ModuleName,
		chronostypes.ModuleName,

		// NOTE: crisis module must go at the end to check for invariants on each module
		crisistypes.ModuleName,
	}
}

func beginBlockerOrder() []string {
	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: upgrade module must go first to handle software upgrades.
	// NOTE: staking module is required if HistoricalEntries param > 0.
	return []string{
		capabilitytypes.ModuleName,
		// Note: epochs' begin should be "real" start of epochs, we keep epochs beginblock at the beginning
		epochstypes.ModuleName,
		feemarkettypes.ModuleName,
		evmtypes.ModuleName,
		chronostypes.ModuleName,
		genutiltypes.ModuleName,
		vestingtypes.ModuleName,
		govtypes.ModuleName,
		hyperiontypes.ModuleName,
		paramstypes.ModuleName,
		authtypes.ModuleName,
		crisistypes.ModuleName,
		feegrant.ModuleName,
		banktypes.ModuleName,
		authz.ModuleName,
		ibctransfertypes.ModuleName,
		consensustypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		packetforwardtypes.ModuleName,
		erc20types.ModuleName,
		tokenfactorytypes.ModuleName,
	}
}

func endBlockerOrder() []string {
	return []string{
		genutiltypes.ModuleName,
		vestingtypes.ModuleName,
		paramstypes.ModuleName,
		authtypes.ModuleName,
		feegrant.ModuleName,
		authz.ModuleName,
		ibctransfertypes.ModuleName,
		consensustypes.ModuleName,
		minttypes.ModuleName,
		slashingtypes.ModuleName,
		ibctransfertypes.ModuleName,
		evidencetypes.ModuleName,
		capabilitytypes.ModuleName,
		distrtypes.ModuleName,
		ibcexported.ModuleName,
		icatypes.ModuleName,
		ibcfeetypes.ModuleName,
		upgradetypes.ModuleName,
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		evmtypes.ModuleName,
		chronostypes.ModuleName,
		feemarkettypes.ModuleName,
		hyperiontypes.ModuleName,
		tokenfactorytypes.ModuleName,
		packetforwardtypes.ModuleName,
		banktypes.ModuleName,
	}
}

// func (app *HeliosApp) setupUpgradeHandlers() {
// 	// v20 upgrade handler
// 	app.UpgradeKeeper.SetUpgradeHandler(
// 		v20.UpgradeName,
// 		v20.CreateUpgradeHandler(
// 			app.mm, app.configurator,
// 			app.AccountKeeper,
// 			app.EvmKeeper,
// 		),
// 	)

// 	// When a planned update height is reached, the old binary will panic
// 	// writing on disk the height and name of the update that triggered it
// 	// This will read that value, and execute the preparations for the upgrade.
// 	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
// 	if err != nil {
// 		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
// 	}

// 	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
// 		return
// 	}

// 	// var storeUpgrades *storetypes.StoreUpgrades

// 	// switch upgradeInfo.Name {
// 	// case v191.UpgradeName:
// 	// 	storeUpgrades = &storetypes.StoreUpgrades{
// 	// 		Added: []string{ratelimittypes.ModuleName},
// 	// 	}
// 	// default:
// 	// // no-op
// 	// }

// 	// if storeUpgrades != nil {
// 	// 	// configure store loader that checks if version == upgradeHeight and applies store upgrades
// 	// 	app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, storeUpgrades))
// 	// }
// }
