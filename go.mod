module helios-core

go 1.23.0

toolchain go1.23.3

require (
	github.com/CosmWasm/wasmd v0.52.0
	github.com/btcsuite/btcd v0.23.4
	github.com/cometbft/cometbft v0.38.10
	github.com/cosmos/cosmos-sdk v0.50.8
	github.com/cosmos/go-bip39 v1.0.0
	github.com/cosmos/gogoproto v1.6.0
	github.com/cosmos/ibc-go/v8 v8.4.0
	github.com/ethereum/go-ethereum v1.11.5
	github.com/golang/protobuf v1.5.4
	github.com/gorilla/mux v1.8.1
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/onsi/ginkgo v1.16.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.8.3
	github.com/spf13/cast v1.7.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.10.0
	github.com/swaggest/swgui v1.4.5
	github.com/tyler-smith/go-bip39 v1.1.0
	golang.org/x/crypto v0.36.0
	golang.org/x/exp v0.0.0-20250106191152-7588d65b2ba8
	google.golang.org/genproto v0.0.0-20240227224415-6ceb2ff114de // indirect
	google.golang.org/grpc v1.64.1
	google.golang.org/protobuf v1.36.4
	gopkg.in/yaml.v2 v2.4.0
)

require (
	cosmossdk.io/api v0.7.5
	cosmossdk.io/client/v2 v2.0.0-beta.1
	cosmossdk.io/core v0.11.1
	cosmossdk.io/errors v1.0.1
	cosmossdk.io/log v1.4.0
	cosmossdk.io/math v1.3.0
	cosmossdk.io/store v1.1.2
	cosmossdk.io/tools/confix v0.1.1
	cosmossdk.io/x/evidence v0.1.1
	cosmossdk.io/x/feegrant v0.1.1
	cosmossdk.io/x/tx v0.13.4
	cosmossdk.io/x/upgrade v0.1.4
	github.com/Helios-Chain-Labs/metrics v0.50.10-helios-2
	github.com/alitto/pond v1.9.2
	github.com/btcsuite/btcd/btcutil v1.1.3
	github.com/cosmos/cosmos-db v1.0.2
	github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v8 v8.0.2
	github.com/cosmos/ibc-apps/modules/rate-limiting/v8 v8.0.0
	github.com/cosmos/ibc-go/modules/capability v1.0.1
	github.com/cosmos/ics23/go v0.10.0
	github.com/cosmos/rosetta v0.50.6
	github.com/creachadair/tomledit v0.0.24
	github.com/dop251/goja v0.0.0-20220405120441-9037c2b61cbf
	github.com/golang/mock v1.6.0
	github.com/hashicorp/go-metrics v0.5.3
	github.com/hashicorp/golang-lru/v2 v2.0.7
	github.com/ledgerwatch/erigon-lib v1.0.0
	github.com/linxGnu/grocksdb v1.9.8
	github.com/onsi/ginkgo/v2 v2.7.0
	github.com/onsi/gomega v1.26.0
	github.com/prometheus/client_golang v1.20.5
	github.com/tidwall/btree v1.7.0
	github.com/tidwall/gjson v1.16.0
	github.com/tidwall/sjson v1.2.5
	github.com/tidwall/wal v1.1.8
	github.com/zbiljic/go-filelock v0.0.0-20170914061330-1dbf7103ab7d
	golang.org/x/image v0.25.0
	golang.org/x/sync v0.12.0
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237
	sigs.k8s.io/yaml v1.4.0
)

require (
	cloud.google.com/go v0.112.1 // indirect
	cloud.google.com/go/compute/metadata v0.3.0 // indirect
	cloud.google.com/go/iam v1.1.6 // indirect
	cloud.google.com/go/storage v1.38.0 // indirect
	cosmossdk.io/collections v0.4.0 // indirect
	cosmossdk.io/depinject v1.0.0 // indirect
	cosmossdk.io/x/circuit v0.1.1 // indirect
	github.com/CosmWasm/wasmvm/v2 v2.1.2 // indirect
	github.com/DataDog/appsec-internal-go v1.5.0 // indirect
	github.com/DataDog/datadog-agent/pkg/obfuscate v0.48.0 // indirect
	github.com/DataDog/datadog-agent/pkg/remoteconfig/state v0.48.1 // indirect
	github.com/DataDog/datadog-go v3.2.0+incompatible // indirect
	github.com/DataDog/go-libddwaf/v2 v2.3.2 // indirect
	github.com/DataDog/go-tuf v1.0.2-0.5.2 // indirect
	github.com/DataDog/gostackparse v0.7.0 // indirect
	github.com/DataDog/sketches-go v1.4.2 // indirect
	github.com/DataDog/zstd v1.5.6 // indirect
	github.com/Helios-Chain-Labs/suplog v0.50.10-helios-2 // indirect
	github.com/PuerkitoBio/purell v1.2.1 // indirect
	github.com/StackExchange/wmi v0.0.0-20180116203802-5d049714c4a6 // indirect
	github.com/VictoriaMetrics/fastcache v1.6.0 // indirect
	github.com/bgentry/go-netrc v0.0.0-20140422174119-9fd32a8b3d3d // indirect
	github.com/bits-and-blooms/bitset v1.13.0 // indirect
	github.com/bool64/dev v0.2.39 // indirect
	github.com/btcsuite/btcd/btcec/v2 v2.3.4 // indirect
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.3.4 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cockroachdb/apd/v2 v2.0.2 // indirect
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/pebble v1.1.4 // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/coinbase/rosetta-sdk-go/types v1.0.0 // indirect
	github.com/cometbft/cometbft-db v1.0.4 // indirect
	github.com/cosmos/gogogateway v1.2.0 // indirect
	github.com/cosmos/rosetta-sdk-go v0.10.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/creachadair/atomicfile v0.3.1 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.2.0 // indirect
	github.com/dgraph-io/badger/v4 v4.5.1 // indirect
	github.com/dgraph-io/ristretto/v2 v2.1.0 // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/dlclark/regexp2 v1.4.1-0.20201116162257-a2a8dda75c91 // indirect
	github.com/ebitengine/purego v0.6.0-alpha.5 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/emicklei/dot v1.6.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/getsentry/sentry-go v0.31.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.1 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/gofrs/uuid v4.4.0+incompatible // indirect
	github.com/gogo/googleapis v1.4.1 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/flatbuffers v25.1.24+incompatible // indirect
	github.com/google/pprof v0.0.0-20230817174616-7a8ec2ada47b // indirect
	github.com/google/s2a-go v0.1.7 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.2 // indirect
	github.com/googleapis/gax-go/v2 v2.12.3 // indirect
	github.com/goware/urlx v0.3.2 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-getter v1.7.5 // indirect
	github.com/hashicorp/go-hclog v1.5.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.5.2 // indirect
	github.com/hashicorp/go-safetemp v1.0.0 // indirect
	github.com/hashicorp/go-version v1.6.0 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/holiman/bloomfilter/v2 v2.0.3 // indirect
	github.com/huandu/skiplist v1.2.0 // indirect
	github.com/iancoleman/orderedmap v0.3.0 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/manifoldco/promptui v0.9.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mixpanel/mixpanel-go v1.2.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasisprotocol/curve25519-voi v0.0.0-20230904125328-1f23a7beb09a // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/oklog/ulid v1.3.1 // indirect
	github.com/outcaste-io/ristretto v0.2.3 // indirect
	github.com/philhofer/fwd v1.1.2 // indirect
	github.com/prometheus/tsdb v0.7.1 // indirect
	github.com/richardartoul/molecule v1.0.1-0.20221107223329-32cfee06a052 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/rjeczalik/notify v0.9.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/secure-systems-lab/go-securesystemslib v0.7.0 // indirect
	github.com/shamaton/msgpack/v2 v2.2.0 // indirect
	github.com/shirou/gopsutil v3.21.4-0.20210419000835-c7a38de76ee5+incompatible // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/tinylru v1.1.0 // indirect
	github.com/tinylib/msgp v1.1.8 // indirect
	github.com/tklauser/go-sysconf v0.3.5 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/ulikunitz/xz v0.5.11 // indirect
	github.com/urfave/cli/v2 v2.27.6 // indirect
	github.com/zondax/ledger-go v0.14.3 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.49.0 // indirect
	go.opentelemetry.io/otel v1.24.0 // indirect
	go.opentelemetry.io/otel/metric v1.24.0 // indirect
	go.opentelemetry.io/otel/trace v1.24.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/oauth2 v0.24.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.31.0 // indirect
	golang.org/x/xerrors v0.0.0-20231012003039-104605ab7028 // indirect
	google.golang.org/api v0.171.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240709173604-40e1e62336c5 // indirect
	gopkg.in/DataDog/dd-trace-go.v1 v1.62.0 // indirect
	gopkg.in/natefinch/npipe.v2 v2.0.0-20160621034901-c1b8fa8bdcce // indirect
	gotest.tools/v3 v3.5.1 // indirect
	pgregory.net/rapid v1.1.0 // indirect
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/99designs/go-keychain v0.0.0-20191008050251-8e49817e8af4 // indirect
	github.com/99designs/keyring v1.2.2 // indirect
	github.com/DataDog/datadog-go/v5 v5.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/alexcesaro/statsd v2.0.0+incompatible // indirect
	github.com/aws/aws-sdk-go v1.44.327 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bgentry/speakeasy v0.1.1-0.20220910012023-760eaf8b6816 // indirect
	github.com/btcsuite/btcd/chaincfg/chainhash v1.0.2 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cosmos/btcutil v1.0.5 // indirect
	github.com/cosmos/cosmos-proto v1.0.0-beta.5
	github.com/cosmos/iavl v1.2.0
	github.com/cosmos/ledger-cosmos-go v0.13.3 // indirect
	github.com/danieljoos/wincred v1.1.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/desertbit/timer v0.0.0-20180107155436-c41aec40b27f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dvsekhvalnov/jose2go v1.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-kit/kit v0.12.0 // indirect
	github.com/go-kit/log v0.2.1 // indirect
	github.com/go-logfmt/logfmt v0.6.0 // indirect
	github.com/godbus/dbus v0.0.0-20190726142602-4481cbc300e2 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.5-0.20220116011046-fa5810519dcb // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/orderedcode v0.0.1 // indirect
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/websocket v1.5.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/gsterjov/go-libsecret v0.0.0-20161001094733-a6f4afe4910c // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.1-vault-5 // indirect
	github.com/hdevalence/ed25519consensus v0.1.0 // indirect
	github.com/holiman/uint256 v1.3.1
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jmhodges/levigo v1.0.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/lib/pq v1.10.7 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mtibben/percent v0.2.1 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/petermattis/goid v0.0.0-20231207134359-e60b3f734c67 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475 // indirect
	github.com/rs/zerolog v1.33.0 // indirect
	github.com/samber/lo v1.50.0
	github.com/sasha-s/go-deadlock v0.3.1 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.1-0.20220721030215-126854af5e6v0d // indirect
	github.com/zondax/hid v0.9.2
	go.etcd.io/bbolt v1.4.0 // indirect
	golang.org/x/net v0.37.0
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	nhooyr.io/websocket v1.8.7 // indirect
)

replace (
	cosmossdk.io/store => github.com/Helios-Chain-Labs/cosmos-sdk/store v0.50.10-helios-127
	// cosmossdk.io/store => ../cosmos-sdk/store
	cosmossdk.io/x/evidence => github.com/Helios-Chain-Labs/cosmos-sdk/x/evidence v0.50.10-helios-126
	cosmossdk.io/x/feegrant => github.com/Helios-Chain-Labs/cosmos-sdk/x/feegrant v0.50.10-helios-126
	cosmossdk.io/x/upgrade => github.com/Helios-Chain-Labs/cosmos-sdk/x/upgrade v0.50.10-helios-126

	github.com/CosmWasm/wasmd => github.com/Helios-Chain-Labs/wasmd v0.50.10-helios-8
	// github.com/CosmWasm/wasmd => ../tests/wasmd
	github.com/bandprotocol/bandchain-packet => github.com/Helios-Chain-Labs/bandchain-packet v0.50.10-helios-1
	github.com/cometbft/cometbft => github.com/Helios-Chain-Labs/cometbft v0.50.10-helios-17
	// github.com/cometbft/cometbft => ../cometbft

	github.com/cosmos/cosmos-sdk => github.com/Helios-Chain-Labs/cosmos-sdk v0.50.10-helios-128
	// github.com/cosmos/cosmos-sdk => ../cosmos-sdk
	github.com/cosmos/ibc-apps/modules/ibc-hooks/v8 => github.com/Helios-Chain-Labs/ibc-apps/modules/ibc-hooks/v8 v8.50.10-helios-1
	github.com/cosmos/ibc-go/v8 => github.com/Helios-Chain-Labs/ibc-go/v8 v8.50.10-helios-40
	// github.com/cosmos/ibc-go/v8 => ../ibc-go
	github.com/ethereum/go-ethereum => github.com/evmos/go-ethereum v1.10.26-evmos-rc4
	github.com/miguelmota/go-ethereum-hdwallet => github.com/Helios-Chain-Labs/go-ethereum-hdwallet v0.50.10-helios-1

	github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7
)
