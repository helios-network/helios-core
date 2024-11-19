---
sidebar_position: 4
title: Join Mainnet
---

# Join Injective Mainnet

## Hardware Specification
Node operators should expect to provision one or more data center locations with redundant power, networking, firewalls, HSMs and servers.

The minimum hardware specifications are as follows, though they might rise as network usage increases:

```
(AWS r6i.2xlarge Instance)
8+ vCPU x64
64+ GB RAM
1+ TB SSD storage
1+ Gbps Network Bandwidth
```

For a more performant node, the following configuration is recommended:

```
(AWS r6i.4xlarge Instance or higher)
16+ vCPU
128+ GB RAM
2+ TB SSD storage
5+ Gbps Network Bandwidth
```

 The more storage allocated, the less frequently data must be pruned from the node. 

## Install `heliades` and `peggo`

See the [Injective chain releases repo](https://github.com/InjectiveLabs/injective-chain-releases/releases/) for the most recent releases. Non-validator node operators do not need to install `peggo`.
```bash
wget https://github.com/InjectiveLabs/injective-chain-releases/releases/download/v1.12.1-1705909076//linux-amd64.zip
unzip linux-amd64.zip
sudo mv peggo /usr/bin
sudo mv heliades /usr/bin
sudo mv libwasmvm.x86_64.so /usr/lib 
```

## Initialize a New Injective Node

Before running Injective node, we need to initialize the chain as well as the node's genesis file:

```bash
# The argument <moniker> is the custom username of your node. It should be human-readable.
export MONIKER=<moniker>
# Injective Mainnet has a chain-id of "injective-1"
heliades init $MONIKER --chain-id injective-1
```

Running the `init` command will create `heliades` default configuration files at `~/.heliades`.

## Prepare Configuration to Join Mainnet

You should now update the default configuration with the Mainnet's genesis file and application config file, as well as configure your persistent peers with seed nodes.
```bash
git clone https://github.com/InjectiveLabs/mainnet-config

# copy genesis file to config directory
cp mainnet-config/10001/genesis.json ~/.heliades/config/genesis.json

# copy config file to config directory
cp mainnet-config/10001/app.toml  ~/.heliades/config/app.toml
```

You can also run verify the checksum of the genesis checksum - 573b89727e42b41d43156cd6605c0c8ad4a1ce16d9aad1e1604b02864015d528
```bash
sha256sum ~/.heliades/config/genesis.json
```

Then update the `seeds` field in `~/.heliades/config/config.toml` with the contents of `mainnet-config/10001/seeds.txt` and update the `timeout_commit` to `300ms`.
```bash
cat mainnet-config/10001/seeds.txt
nano ~/.heliades/config/config.toml
```

## Configure `systemd` Service for `heliades`

Edit the config at `/etc/systemd/system/heliades.service`:
```bash
[Unit]
  Description=heliades

[Service]
  WorkingDirectory=/usr/bin
  ExecStart=/bin/bash -c '/usr/bin/heliades --log-level=error start'
  Type=simple
  Restart=always
  RestartSec=5
  User=root

[Install]
  WantedBy=multi-user.target
```

Starting and restarting the systemd service:
```bash
sudo systemctl daemon-reload
sudo systemctl restart heliades
sudo systemctl status heliades

# enable start on system boot
sudo systemctl enable heliades

# To check Logs
journalctl -u heliades -f
```

The service should be stopped before and started after the snapshot data has been loaded into the correct directory.
```bash
# to stop the node
sudo systemctl stop heliades

# to start the node
sudo systemctl start heliades
```

## Sync with the network

### Option 1. State-Sync

*To be added soon*

[//]: # (You can use state-sync to join the network by following the below instructions. Note that the `wasm` directory of the `heliades` configuration files will not be synced and must be updated from the snapshot.)

[//]: # (```bash)

[//]: # (#!/bin/bash)

[//]: # (sudo systemctl stop heliades)

[//]: # (sudo heliades tendermint unsafe-reset-all --home ~/.heliades)

[//]: # (CUR_HEIGHT=$&#40;curl -sS https://tm.helios.network/block | jq .result.block.header.height | tr -d '"'&#41;)

[//]: # (SNAPSHOT_INTERVAL=1000)

[//]: # (RPC_SERVERS="23d0eea9bb42316ff5ea2f8b4cd8475ef3f35209\@65.109.36.70:11750,38c18461209694e1f667ff2c8636ba827cc01c86\@176.9.143.252:11750,4f9025feca44211eddc26cd983372114947b2e85\@176.9.140.49:11750,c98bb1b889ddb58b46e4ad3726c1382d37cd5609\@65.109.51.80:11750,f9ae40fb4a37b63bea573cc0509b4a63baa1a37a\@15.235.144.80:11750,7f3473ddab10322b63789acb4ac58647929111ba\@15.235.13.116:11750")

[//]: # (TRUST_HEIGHT=$&#40;&#40; CUR_HEIGHT - SNAPSHOT_INTERVAL &#41;&#41;)

[//]: # (TRUSTED_HASH=$&#40;curl -sS https://tm.helios.network/block?height=$TRUST_HEIGHT | jq .result.block_id.hash&#41;)

[//]: # (perl -i -pe 's|enable = false|enable = true|g' ~/.heliades/config/config.toml)

[//]: # (perl -i -pe 's|rpc_servers = ".*?"|rpc_servers = "'$RPC_SERVERS'"|g' ~/.heliades/config/config.toml)

[//]: # (perl -i -pe 's/^trust_height = \d+/trust_height = '$TRUST_HEIGHT'/' ~/.heliades/config/config.toml)

[//]: # (perl -i -pe 's/^trust_hash = ".*?"/trust_hash = '$TRUSTED_HASH'/' ~/.heliades/config/config.toml)

[//]: # (sudo systemctl start heliades)

[//]: # (```)

### Option 2. Snapshots

**Pruned**

1. [Polkachu](https://polkachu.com/tendermint_snapshots/injective).
2. [HighStakes](https://tools.highstakes.ch/files/helios.tar.gz).
3. [AutoStake](http://snapshots.autostake.net/injective-1/).
4. [Imperator](https://www.imperator.co/services/chain-services/injective).
5. [Bware Labs](https://bwarelabs.com/snapshots).

Alternatively, you can use the pruned snapshots from Injective Labs on AWS S3.

```bash
systemctl stop heliades
heliades tendermint unsafe-reset-all --home $HOME/.heliades
SNAP=$(aws s3 ls --no-sign-request s3://injective-snapshots/mainnet/pruned/ | grep ".tar.lz4" | sort | tail -n 1 | awk '{print $4}')
aws s3 cp --no-sign-request s3://injective-snapshots/mainnet/pruned/$SNAP .
lz4 -c -d $SNAP  | tar -x -C $HOME/.heliades/
rm $SNAP
systemctl start heliades
```


Should the Injective `mainnet-config seeds.txt` list not work (the node fails to sync blocks), ChainLayer, Polkachu, and Autostake maintain peer lists (can be used in the `persistent_peers` field in `config.toml`) or addressbooks (for faster peer discovery).

**Archival** (>20TB)

```bash
systemctl stop heliades
heliades tendermint unsafe-reset-all --home $HOME/.heliades
aws s3 sync --no-sign-request --delete s3://injective-snapshots/mainnet/heliades/data $HOME/.heliades/data
aws s3 sync --no-sign-request --delete s3://injective-snapshots/mainnet/heliades/wasm $HOME/.heliades/wasm
systemctl start heliades
```

At this point, [GEX](https://github.com/cosmos/gex) can be used to monitor the node's sync status. If the snapshot has been correcly loaded, the number of connected peers should increase from 0 and the latest block should steadily increase, signalling the node syncing with its peers. Note that it may take a few or several hours for the node to catch up to the network's block height depending on the age of the snapshot.

In the case where the latest block does not increase and the number of connected peers is 0 or remains low, the seed list in `seeds.txt` may be outdated, and the `seeds` or `persistent_peers` fields can be updated using a validator's seed or peer list respectively, before the node is started again.
```bash
go install github.com/cosmos/gex@latest
gex
```

### Support

For any further questions, you can always connect with the Injective Team via [Discord](https://discord.gg/injective), [Telegram](https://t.me/joininjective), or [email](mailto:contact@injectivelabs.org)

