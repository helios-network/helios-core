# Update configuration files
perl -i -pe 's/^timeout_commit = ".*?"/timeout_commit = "5000ms"/' ~/.heliades/config/config.toml

# Minimum gas Price accepted by the current node validator
perl -i -pe 's/^minimum-gas-prices = ".*?"/minimum-gas-prices = "0.1ahelios"/' ~/.heliades/config/app.toml

# Update genesis file with new denominations and parameters
GENESIS_CONFIG="$HOME/.heliades/config/genesis.json"
TMP_GENESIS="$HOME/.heliades/config/tmp_genesis.json"

jq '.app_state["gov"]["params"]["min_initial_deposit_ratio"]="0.100000000000000000"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
echo "NOTE: Setting Governance Voting Period to 10 seconds for easy testing"
jq '.app_state["gov"]["params"]["voting_period"]="30s"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
jq '.app_state["gov"]["params"]["expedited_voting_period"]="10s"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
jq '.app_state["staking"]["params"]["unbonding_time"]="5s"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# Set gas limit in genesis
jq '.consensus_params["block"]["max_gas"]="10000000"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# Set base fee at the start of the blockchain (1Gwei = 1000000000)
jq '.app_state["feemarket"]["params"]["base_fee"]="1000000000"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG
# Set min_gas_price it's the minmum base fee possible (1Gwei = 1000000000) 
jq '.app_state["feemarket"]["params"]["min_gas_price"]="1000000000"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG

# HELIOS DEFAULT COMMUNITY TAX
jq '.app_state["distribution"]["params"]["community_tax"]="0.02"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG

#DEFAULT STAKE OVERALL DISTRIB MANAGEMENT
#- `dominance_threshold`: Percentage threshold above which reduction begins
#- `curve_steepness`: Controls how quickly reduction increases
#- `max_reduction`: Maximum possible reduction percentage
#- `enabled`: Whether the mechanism is active

# jq '.app_state["staking"]["params"]["delegator_stake_reduction"] |= 
#     { 
#       "enabled": true, 
#       "dominance_threshold": "0.05",
#       "max_reduction": "0.90",
#       "curve_steepness": "10.0"
#     }' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG


# Zero address account (burn)
#jq '.app_state.bank.balances += [{"address": "helios1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqe2hm49", "coins": [{"denom": "helios", "amount": "1"}]}]' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG

# Add genesis accounts
TREASURY_ADDRESS="helios1aj2gcctecp874q90jclsuk6c2k6kvdthwek60l" # DO NOT FORGET TO UPDATE FOR TESTNET/MAINNET!!!

# Save as treasury wallet
jq '.app_state["staking"]["params"]["treasury_address"]="'$TREASURY_ADDRESS'"' $GENESIS_CONFIG > $TMP_GENESIS && mv $TMP_GENESIS $GENESIS_CONFIG