#!/bin/bash

make install

yes | ./setup.sh

# ./heliades.sh > /dev/null 2>&1 &

# sleep 3s
sleep 3

heliades tx hyperion set-orchestrator-address helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf 0x17267eB1FEC301848d4B5140eDDCFC48945427Ab --chain-id=4242 --node="tcp://localhost:26657" --from="helios1zun8av07cvqcfr2t29qwmh8ufz29gfatfue0cf" --gas-prices=600000000ahelios --yes