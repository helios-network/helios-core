---
sidebar_position: 3
title: Join Testnet
---

# Join the Network

## Hardware Specification
Node operators should expect to provision one or more data center locations with redundant power, networking, firewalls, HSMs and servers.

The minimum hardware specifications are as follows, though they might rise as network usage increases:

```
6+ vCPU x64
32+ GB RAM
500 GB+ SSD
```

## Install `heliades` and `peggo`

See the [Injective releases repo](https://github.com/InjectiveLabs/testnet/releases) for the most recent releases. Non-validator node operators do not need to install `peggo`.

```bash
wget https://github.com/InjectiveLabs/testnet/releases/download/v1.12.9-testnet-1703762556/linux-amd64.zip
unzip linux-amd64.zip
sudo mv peggo /usr/bin
sudo mv heliades /usr/bin
sudo mv libwasmvm.x86_64.so /usr/lib 
```

## Initialize a New Injective Chain Node

Before running Injective node, we need to initialize the chain as well as the node's genesis file:

```bash
# The argument <moniker> is the custom username of your node, it should be human-readable.
export MONIKER=<moniker>
# Injective Testnet has a chain-id of "injective-888"
heliades init $MONIKER --chain-id injective-888
```

Running the `init` command will create `heliades` default configuration files at `~/.heliades`.

## Prepare Configuration to Join Testnet

You should now update the default configuration with the Testnet's genesis file and application config file, as well as configure your persistent peers with seed nodes.

```bash
git clone https://github.com/InjectiveLabs/testnet.git

# copy genesis file to config directory
aws s3 cp --no-sign-request s3://injective-snapshots/testnet/genesis.json .
mv genesis.json ~/.heliades/config/

# copy config file to config directory
cp testnet/corfu/70001/app.toml  ~/.heliades/config/app.toml
cp testnet/corfu/70001/config.toml ~/.heliades/config/config.toml
```

You can also run verify the checksum of the genesis checksum - a4abe4e1f5511d4c2f821c1c05ecb44b493eec185c0eec13b1dcd03d36e1a779
```bash
sha256sum ~/.heliades/config/genesis.json
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

Starting and restarting the systemd service
```bash
sudo systemctl daemon-reload
sudo systemctl restart heliades
sudo systemctl status heliades

# enable start on system boot
sudo systemctl enable heliades

# To check Logs
journalctl -u heliades -f
```

## Sync with the network

Refer to the Polkachu guide [here](https://polkachu.com/testnets/injective/snapshots) to download a snapshot and sync with the network.


### Support

For any further questions, you can always connect with the Injective Team via [Discord](https://discord.gg/injective), [Telegram](https://t.me/joininjective), or [email](mailto:contact@injectivelabs.org).
