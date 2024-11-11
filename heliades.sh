#!/bin/bash
ulimit -n 120000
yes 12345678 | heliades start \
--log-level "info" \
--rpc.laddr "tcp://0.0.0.0:26657" \
--api.address="tcp://0.0.0.0:1317" \
--minimum-gas-prices "0.1helios" \
--grpc.enable=true --grpc.address="0.0.0.0:9090"