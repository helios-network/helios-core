#!/bin/bash

BOOTSTRAP_RPC="http://node1:26657"
NODE_NAME=${NODE_NAME:-unknown}

if [ "$NODE_NAME" != "node1" ]; then
    echo "Waiting for node1 to be ready..."
    while ! curl -s "$BOOTSTRAP_RPC/status" > /dev/null; do
        echo "Node1 not ready yet. Retrying in 5 seconds..."
        sleep 5
    done

    echo "Node1 is ready. Configuring persistent peers..."

    # Get the ID of node1
    BOOTSTRAP_ID=$(curl -s "$BOOTSTRAP_RPC/status" | jq -r '.result.node_info.id')

    if [ -n "$BOOTSTRAP_ID" ]; then
        # Update persistent_peers in config.toml
        sed -i "s/^persistent_peers *=.*/persistent_peers = \"$BOOTSTRAP_ID@node1:26656\"/" /root/.heliades/config/config.toml
        echo "Persistent peers configured: $BOOTSTRAP_ID@node1:26656"
    else
        echo "Error: Failed to retrieve BOOTSTRAP_ID"
        exit 1
    fi

    echo "Copying genesis.json from node1 to current node..."
    curl -s "$BOOTSTRAP_RPC/genesis" | jq '.result.genesis' > /root/.heliades/config/genesis.json

    if [ $? -eq 0 ]; then
        echo "Genesis file replaced successfully."
    else
        echo "Error: Failed to copy genesis.json from node1."
        exit 1
    fi

    echo "Updating app.toml with BOOTSTRAP_RPC..."
    sed -i "/^# RPC-related configuration$/a \
rpc_servers = \"$BOOTSTRAP_RPC,$BOOTSTRAP_RPC\"\n\
rpc_id = \"$(curl -s $BOOTSTRAP_RPC/status | jq -r '.result.node_info.id')\"\n\
" /root/.heliades/config/app.toml

    if [ $? -eq 0 ]; then
        echo "app.toml updated successfully."
    else
        echo "Error: Failed to update app.toml."
        exit 1
    fi

else
    echo "This is the bootstrap node (node1), skipping peer configuration."
fi

echo "Starting node $NODE_NAME..."

ulimit -n 120000
yes 12345678 | heliades start \
--chain-id 4242 \
--log_level "info" \
--rpc.laddr "tcp://0.0.0.0:26657" \
--minimum-gas-prices "0.1helios" \
--grpc.enable=true --grpc.address="0.0.0.0:9090" \
--json-rpc.enable=true \
--json-rpc.api eth,txpool,personal,net,debug,web3 \
--json-rpc.address="0.0.0.0:8545"
