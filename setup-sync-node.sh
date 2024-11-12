#!/bin/bash

set -e

# Build and install the heliades binary
#make install

# Stop any running instances of heliades and clean up old data
killall heliades &>/dev/null || true
rm -rf ~/.heliades

# Define chain parameters
CHAINID="4242"
# Name of your node
MONIKER="helios1"

# Initialize the chain with a moniker and chain ID
heliades init $MONIKER --chain-id $CHAINID

# Update configuration files
perl -i -pe 's/^timeout_commit = ".*?"/timeout_commit = "2500ms"/' ~/.heliades/config/config.toml
perl -i -pe 's/^minimum-gas-prices = ".*?"/minimum-gas-prices = "500000000helios"/' ~/.heliades/config/app.toml

# Update genesis file with new denominations and parameters
GENESIS_CONFIG="$HOME/.heliades/config/genesis.json"
TMP_GENESIS="$HOME/.heliades/config/tmp_genesis.json"

# Setup sync

# RPC url of your trusted node
SNAP_RPC="http://192.168.1.75:26657"
RPC_ID=$(curl -s $SNAP_RPC/status | jq -r .result.node_info.id);
# IP of your trusted node
RPC_IP="192.168.1.75"

sed -i.bak -E "s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$SNAP_RPC,$SNAP_RPC\"| ; \
 s|^(persistent_peers[[:space:]]+=[[:space:]]+).*$|\1\"$RPC_ID@$RPC_IP:26656\"|" ~/.heliades/config/config.toml


## Specifical metrics for creating a lite node:

# LATEST_HEIGHT=$(curl -s $SNAP_RPC/block | jq -r .result.block.header.height); \
# BLOCK_HEIGHT=$((LATEST_HEIGHT - 100)); \
# TRUST_HASH=$(curl -s "$SNAP_RPC/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

# sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true| ; \
# s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$SNAP_RPC,$SNAP_RPC\"| ; \
# s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$BLOCK_HEIGHT| ; \
# s|^(persistent_peers[[:space:]]+=[[:space:]]+).*$|\1\"$RPC_ID@$RPC_IP:26656\"| ; \
# s|^(trust_period[[:space:]]+=[[:space:]]+).*$|\1\"1h0m0s\"| ; \
# s|^(chunk_fetchers[[:space:]]+=[[:space:]]+).*$|\1\"1\"| ; \
# s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.heliades/config/config.toml

# perl -i -pe 's/^pruning = ".*?"/pruning = "custom"/' ~/.heliades/config/app.toml
# perl -i -pe 's/^pruning-interval = ".*?"/pruning-interval = "100"/' ~/.heliades/config/app.toml
# perl -i -pe 's/^pruning-keep-recent = ".*?"/pruning-keep-recent = "5"/' ~/.heliades/config/app.toml


## Specifical metrics for enabling a snapshot node
# perl -i -pe 's/^snapshot-interval = \d+/snapshot-interval = 100/' ~/.heliades/config/app.toml
# perl -i -pe 's/^snapshot-keep-recent = \d+/snapshot-keep-recent = 5/' ~/.heliades/config/app.toml