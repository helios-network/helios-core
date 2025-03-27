#!/usr/bin/env bash

set -eo pipefail

SWAGGER_TMP_DIR=tmp-swagger-gen
SWAGGER_BUILD_DIR=tmp-swagger-build
COSMOS_SDK_VERSION_TAG=v0.50.10-helios-64
IBC_GO_VERSION_TAG=v8.50.10-helios-1
WASMD_VERSION_TAG=v0.50.6-helios-3
COMETBFT_VERSION_TAG=v0.50.10-helios-11
rm -fr $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR
mkdir -p $SWAGGER_BUILD_DIR $SWAGGER_TMP_DIR

cd $SWAGGER_BUILD_DIR
mkdir -p proto
printf "version: v1\ndirectories:\n  - proto\n  - third_party" > buf.work.yaml
printf "version: v1\nname: buf.build/helios-network/helios-core\n" > proto/buf.yaml
cp ../proto/buf.gen.swagger.yaml proto/buf.gen.swagger.yaml
cp -r ../proto/helios proto/
cp -r ../proto/ethermint proto/

# Clone repositories
git clone https://github.com/helios-network/cosmos-sdk.git -b $COSMOS_SDK_VERSION_TAG --depth 1 --single-branch
git clone https://github.com/helios-network/ibc-go.git -b $IBC_GO_VERSION_TAG --depth 1 --single-branch
git clone https://github.com/helios-network/wasmd.git -b $WASMD_VERSION_TAG --depth 1 --single-branch
git clone https://github.com/helios-network/cometbft.git -b $COMETBFT_VERSION_TAG --depth 1 --single-branch

buf export ./cosmos-sdk --output=third_party
buf export ./ibc-go --exclude-imports --output=third_party
buf export ./wasmd --exclude-imports --output=./third_party
buf export ./cometbft --exclude-imports --output=./third_party
buf export https://github.com/cosmos/ics23.git --exclude-imports --output=./third_party

# Modified IBC apps export and directory structure
mkdir -p ./third_party/packetforward
git clone --depth 1 https://github.com/cosmos/ibc-apps.git
cp -r ./ibc-apps/middleware/packet-forward-middleware/proto/packetforward ./third_party/
rm -rf ./ibc-apps

# Generate swagger files
proto_dirs=$(find ./proto ./third_party -type f -name '*.proto' -exec dirname {} \; | sort | uniq)
for dir in $proto_dirs; do
  query_file=$(find "$dir" -maxdepth 1 -name 'query.proto' -o -name 'service.proto' )
  if [ -n "$query_file" ]; then
    echo "generating $query_file"
    buf generate --template proto/buf.gen.swagger.yaml "$query_file"
  fi
done

echo "Generated swagger files"

# Create directory structure for swagger files
mkdir -p ../$SWAGGER_TMP_DIR/packetforward/v1

# Copy generated swagger files to expected location
if [ -f "./third_party/packetforward/v1/query.swagger.json" ]; then
    cp ./third_party/packetforward/v1/query.swagger.json ../$SWAGGER_TMP_DIR/packetforward/v1/
fi

rm -rf ./cosmos-sdk && rm -rf ./ibc-go && rm -rf ./wasmd

cd ..
echo "Combining swagger files"

# Ensure the directory exists before combining
mkdir -p ./client/docs/swagger-ui

swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

echo "Cleaning up"

rm -rf $SWAGGER_TMP_DIR $SWAGGER_BUILD_DIR

echo "Convert swagger.yaml to openapi.json"

swagger-cli bundle ./client/docs/swagger-ui/swagger.yaml --outfile ./client/docs/swagger-ui/openapi.json --type json

cp ./client/docs/swagger-ui/openapi.json ./helios-chain/server/grpc-openapi.json