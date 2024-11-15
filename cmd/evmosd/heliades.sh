#!/bin/bash
ulimit -n 120000
yes 12345678 | ./evmosd start \
--chain-id 4242 \
--log_level "info" \
--rpc.laddr "tcp://0.0.0.0:26657" \
--minimum-gas-prices "0.1helios" \
--grpc.enable=true --grpc.address="0.0.0.0:9090" \
--json-rpc.api eth,txpool,personal,net,debug,web3 \
--json-rpc.address="0.0.0.0:8545"