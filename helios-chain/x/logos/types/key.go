package types

const (
	// ModuleName is the name of the module
	ModuleName = "logos"

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
	ParamsKey = []byte{0x1}

	// LogoKey indexes logos by hash
	LogoKey = []byte{0x2}
)
