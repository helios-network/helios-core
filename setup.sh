#!/bin/bash

set -e

# Build and install the heliades binary
# make install

# Stop any running instances of heliades and clean up old data
killall heliades &>/dev/null || true
rm -rf ~/.heliades

# Define chain parameters
CHAINID="4242"
MONIKER="helios"
PASSPHRASE="12345678"
FEEDADMIN="helios1q0d2nv8xpf9qy22djzgrkgrrcst9frcs34fqra"

# Initialize the chain with a moniker and chain ID
heliades init $MONIKER --chain-id $CHAINID

# echo '[json-rpc]' >> ~/.heliades/config/app.toml
# echo '# Enable defines if the JSON-RPC server should be enabled.' >> ~/.heliades/config/app.toml
# echo 'enable = true' >> ~/.heliades/config/app.toml
# echo '# Address defines the JSON-RPC server address to bind to.' >> ~/.heliades/config/app.toml
# echo 'address = "0.0.0.0:8545"' >> ~/.heliades/config/app.toml
# echo '# API defines a list of JSON-RPC namespaces that should be enabled' >> ~/.heliades/config/app.toml
# echo 'api = ["eth","txpool","personal","net","debug","web3"]' >> ~/.heliades/config/app.toml

# Update configuration files
# perl -i -pe 's/^timeout_commit = ".*?"/timeout_commit = "2500ms"/' ~/.heliades/config/config.toml
# perl -i -pe 's/^minimum-gas-prices = ".*?"/minimum-gas-prices = "500000000helios"/' ~/.heliades/config/app.toml

# # Update genesis file with new denominations and parameters
# GENESIS_CONFIG="$HOME/.heliades/config/genesis.json"
# TMP_GENESIS="$HOME/.heliades/config/tmp_genesis.json"

# jq '.app_state["staking"]["params"]["bond_denom"]="helios"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["crisis"]["constant_fee"]["denom"]="helios"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["gov"]["params"]["min_deposit"][0]["denom"]="helios"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["gov"]["params"]["min_initial_deposit_ratio"]="0.100000000000000000"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# echo "NOTE: Setting Governance Voting Period to 10 seconds for easy testing"
# jq '.app_state["gov"]["params"]["voting_period"]="10s"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["gov"]["params"]["expedited_voting_period"]="5s"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["mint"]["params"]["mint_denom"]="helios"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["auction"]["params"]["auction_period"]="10"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["ocr"]["params"]["module_admin"]="'$FEEDADMIN'"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# jq '.app_state["ocr"]["params"]["payout_block_interval"]="5"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# # Zero address account (burn)
# #jq '.app_state.bank.balances += [{"address": "helios1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqe2hm49", "coins": [{"denom": "helios", "amount": "1"}]}]' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG

# # Define token denominations and decimals
# HELIOS='{"denom":"helios","decimals":18}'


# USDT='{"denom":"peggy0xdAC17F958D2ee523a2206206994597C13D831ec7","decimals":6}'
# USDC='{"denom":"peggy0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48","decimals":6}'
# ONEINCH='{"denom":"peggy0x111111111117dc0aa78b770fa6a738034120c302","decimals":18}'
# AAVE='{"denom":"peggy0x7fc66500c84a76ad7e9c93437bfc5ac33e2ddae9","decimals":18}'
# AXS='{"denom":"peggy0xBB0E17EF65F82Ab018d8EDd776e8DD940327B28b","decimals":18}'
# BAT='{"denom":"peggy0x0D8775F648430679A709E98d2b0Cb6250d2887EF","decimals":18}'
# BNB='{"denom":"peggy0xB8c77482e45F1F44dE1745F52C74426C631bDD52","decimals":18}'
# WBTC='{"denom":"peggy0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599","decimals":8}'
# BUSD='{"denom":"peggy0x4Fabb145d64652a948d72533023f6E7A623C7C53","decimals":18}'
# CELL='{"denom":"peggy0x26c8AFBBFE1EBaca03C2bB082E69D0476Bffe099","decimals":18}'
# CHZ='{"denom":"peggy0x3506424F91fD33084466F402d5D97f05F8e3b4AF","decimals":18}'
# COMP='{"denom":"peggy0xc00e94Cb662C3520282E6f5717214004A7f26888","decimals":18}'
# DAI='{"denom":"peggy0x6B175474E89094C44Da98b954EedeAC495271d0F","decimals":18}'
# DEFI5='{"denom":"peggy0xfa6de2697D59E88Ed7Fc4dFE5A33daC43565ea41","decimals":18}'
# ENJ='{"denom":"peggy0xF629cBd94d3791C9250152BD8dfBDF380E2a3B9c","decimals":18}'
# WETH='{"denom":"peggy0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2","decimals":18}'
# EVAI='{"denom":"peggy0x50f09629d0afDF40398a3F317cc676cA9132055c","decimals":8}'
# FTM='{"denom":"peggy0x4E15361FD6b4BB609Fa63C81A2be19d873717870","decimals":18}'
# GF='{"denom":"peggy0xAaEf88cEa01475125522e117BFe45cF32044E238","decimals":18}'
# GRT='{"denom":"peggy0xc944E90C64B2c07662A292be6244BDf05Cda44a7","decimals":18}'
# HT='{"denom":"peggy0x6f259637dcD74C767781E37Bc6133cd6A68aa161","decimals":18}'
# LINK='{"denom":"peggy0x514910771AF9Ca656af840dff83E8264EcF986CA","decimals":18}'
# MATIC='{"denom":"peggy0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0","decimals":18}'
# NEXO='{"denom":"peggy0xB62132e35a6c13ee1EE0f84dC5d40bad8d815206","decimals":18}'
# NOIA='{"denom":"peggy0xa8c8CfB141A3bB59FEA1E2ea6B79b5ECBCD7b6ca","decimals":18}'
# OCEAN='{"denom":"peggy0x967da4048cD07aB37855c090aAF366e4ce1b9F48","decimals":18}'
# PAXG='{"denom":"peggy0x45804880De22913dAFE09f4980848ECE6EcbAf78","decimals":18}'
# POOL='{"denom":"peggy0x0cEC1A9154Ff802e7934Fc916Ed7Ca50bDE6844e","decimals":18}'
# QNT='{"denom":"peggy0x4a220E6096B25EADb88358cb44068A3248254675","decimals":18}'
# RUNE='{"denom":"peggy0x3155BA85D5F96b2d030a4966AF206230e46849cb","decimals":18}'
# SHIB='{"denom":"peggy0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE","decimals":18}'
# SNX='{"denom":"peggy0xC011a73ee8576Fb46F5E1c5751cA3B9Fe0af2a6F","decimals":18}'
# STARS='{"denom":"peggy0xc55c2175E90A46602fD42e931f62B3Acc1A013Ca","decimals":18}'
# STT='{"denom":"peggy0xaC9Bb427953aC7FDDC562ADcA86CF42D988047Fd","decimals":18}'
# SUSHI='{"denom":"peggy0x6B3595068778DD592e39A122f4f5a5cF09C90fE2","decimals":18}'
# SWAP='{"denom":"peggy0xCC4304A31d09258b0029eA7FE63d032f52e44EFe","decimals":18}'
# UMA='{"denom":"peggy0x04Fa0d235C4abf4BcF4787aF4CF447DE572eF828","decimals":18}'
# UNI='{"denom":"peggy0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984","decimals":18}'
# UTK='{"denom":"peggy0xdc9Ac3C20D1ed0B540dF9b1feDC10039Df13F99c","decimals":18}'
# YFI='{"denom":"peggy0x0bc529c00C6401aEF6D220BE8C6Ea1667F6Ad93e","decimals":18}'
# ZRX='{"denom":"peggy0xE41d2489571d322189246DaFA5ebDe1F4699F498","decimals":18}'

# ATOM='{"denom":"ibc/C4CFF46FD6DE35CA4CF4CE031E643C8FDC9BA4B99AE598E9B0ED98FE3A2319F9","decimals":6}'
# USTC='{"denom":"ibc/B448C0CA358B958301D328CCDC5D5AD642FC30A6D3AE106FF721DB315F3DDE5C","decimals":6}'
# AXL='{"denom":"ibc/C49B72C4E85AE5361C3E0F0587B24F509CB16ECEB8970B6F917D697036AF49BE","decimals":6}'
# XPRT='{"denom":"ibc/B786E7CBBF026F6F15A8DA248E0F18C62A0F7A70CB2DABD9239398C8B5150ABB","decimals":6}'
# SCRT='{"denom":"ibc/3C38B741DF7CD6CAC484343A4994CFC74BC002D1840AAFD5416D9DAC61E37F10","decimals":6}'
# OSMO='{"denom":"ibc/92E0120F15D037353CFB73C14651FC8930ADC05B93100FD7754D3A689E53B333","decimals":6}'
# LUNC='{"denom":"ibc/B8AF5D92165F35AB31F3FC7C7B444B9D240760FA5D406C49D24862BD0284E395","decimals":6}'
# HUAHUA='{"denom":"ibc/E7807A46C0B7B44B350DA58F51F278881B863EC4DCA94635DAB39E52C30766CB","decimals":6}'
# EVMOS='{"denom":"ibc/16618B7F7AC551F48C057A13F4CA5503693FBFF507719A85BC6876B8BD75F821","decimals":18}'
# DOT='{"denom":"ibc/624BA9DD171915A2B9EA70F69638B2CEA179959850C1A586F6C485498F29EDD4","decimals":10}'



# # Update the list of tokens with their denominations and decimals
# PEGGY_DENOM_DECIMALS="${USDT},${USDC},${ONEINCH},${AXS},${BAT},${BNB},${WBTC},${BUSD},${CELL},${CHZ},${COMP},${DAI},${DEFI5},${ENJ},${WETH},${EVAI},${FTM},${GF},${GRT},${HT},${LINK},${MATIC},${NEXO},${NOIA},${OCEAN},${PAXG},${POOL},${QNT},${RUNE},${SHIB},${SNX},${STARS},${STT},${SUSHI},${SWAP},${UMA},${UNI},${UTK},${YFI},${ZRX}"
# IBC_DENOM_DECIMALS="${ATOM},${USTC},${AXL},${XPRT},${SCRT},${OSMO},${LUNC},${HUAHUA},${EVMOS},${DOT}"
# DENOM_DECIMALS='['${HELIOS},${PEGGY_DENOM_DECIMALS},${IBC_DENOM_DECIMALS}']'
# #DENOM_DECIMALS='['${HELIOS}']'

# jq '.app_state["exchange"]["denom_decimals"]='${DENOM_DECIMALS} $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG

# Add genesis accounts
yes $PASSPHRASE | heliades keys add genesis
yes $PASSPHRASE | heliades add-genesis-account --chain-id $CHAINID $(yes $PASSPHRASE | heliades keys show genesis -a) 1000000000000000000000000helios

# Define the keys array
KEYS=(
    "wasm"
    "localkey"
    "user1"
    "user2"
    "user3"
    "user4"
    "ocrfeedadmin"
    "signer1"
    "signer2"
    "signer3"
    "signer4"
    "signer5"
)


# Define mnemonics array (in same order as keys)
MNEMONICS=(
"drill rabbit course stay climb inch later primary ghost seat demise lava cry later struggle mountain segment near wagon wood used fashion budget neglect"
"run fetch fantasy stairs explain transfer sweet goat negative cliff fetch awake sense regular roof stool worth empty trip salute account wave certain devote"
"west mouse extra original dizzy dinosaur corn lottery access off slab surge piano build rabbit educate amused trophy orbit cable relax chimney trend inner"
"final rude almost banner language raven soon world aim pole copper poverty camera post wet humble hurt element find alone frown damp feature sadness"
"pupil target orient whip evidence life uniform mother senior strong lizard lens gesture young east armor loop library shadow fee host throw junk more"
"museum violin scan lonely knife fiscal ask science treat undo vacuum mention surge uniform mail tackle cricket artwork mimic alpha hero north before loan"
"silver rack profit either powder ridge copy memory awkward exit name wink heart cherry antenna talent derive topple caution second survey dream angle salute"
"lawsuit fame nice soft left method source ticket stage tourist unfold audit often reveal raise okay project absorb bubble spoon bounce track ready poet"
"slow fine dentist give small black shrug mouse fix coral omit type fish palace portion rhythm danger cream notice bless print pioneer announce course"
"liar oven damp useless again please dream birth box bottom hat olive slow rice busy atom carpet pilot always trust balcony hammer extend laptop"
"cash shoulder people eternal expire occur pen black funny idle afraid manual pause replace faith goose junior kite forest poet pulp treat cable merry"
"ostrich prefer glad boat slight hedgehog burden manage enforce post wrap pottery daring delay video energy mammal urge enemy prevent wool badge garage thrive"
)

# Import keys from mnemonics
for i in "${!KEYS[@]}"; do
    printf "${MNEMONICS[$i]}\n$PASSPHRASE" | heliades keys add ${KEYS[$i]} --recover
done

# Allocate genesis accounts
for key in "${KEYS[@]}"; do
    printf $PASSPHRASE | heliades add-genesis-account --chain-id $CHAINID $(heliades keys show $key -a) 1000000000000000000000helios
done

echo "Signing genesis transaction"
# Sign genesis transaction
#yes $PASSPHRASE | heliades genesis gentx genesis 1000000000000000000000helios --chain-id $CHAINID
yes $PASSPHRASE | heliades gentx genesis 1000000000000000000000helios --chain-id $CHAINID

echo "Collecting genesis transaction"
# Collect genesis tx
# yes $PASSPHRASE | heliades genesis collect-gentxs
yes $PASSPHRASE | heliades collect-gentxs

echo "Validating genesis"
# Validate the genesis file
# heliades genesis validate
heliades validate-genesis

echo "Setup done!"