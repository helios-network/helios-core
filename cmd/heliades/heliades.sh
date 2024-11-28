#!/bin/bash
ulimit -n 120000
./heliades start \
--chain-id 4242 \
--log_level "info" \
--rpc.laddr "tcp://0.0.0.0:26657" \
--minimum-gas-prices "0.1ahelios" \
--grpc.enable=true --grpc.address="0.0.0.0:9090" \
--json-rpc.api eth,txpool,personal,net,debug,web3 \
--json-rpc.address="0.0.0.0:8545"