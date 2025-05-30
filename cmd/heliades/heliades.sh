#!/bin/bash
ulimit -n 120000
./heliades start \
--chain-id 4242 \
--log_level "info" \
--rpc.laddr "tcp://0.0.0.0:26657" \
--minimum-gas-prices "0.1helios" \
--grpc.enable=true \
--grpc.address="0.0.0.0:9090" \
--grpc-web.enable=true \
--api.enable=true \
--api.enabled-unsafe-cors=true \
--json-rpc.enable=true \
--json-rpc.api "eth,txpool,personal,net,debug,web3" \
--json-rpc.address "0.0.0.0:8545" \
--json-rpc.ws-address "0.0.0.0:8546" \
--p2p.laddr "tcp://0.0.0.0:26656"
