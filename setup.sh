#!/bin/bash

set -e

# Stop any running instances of heliades and clean up old data
killall heliades &>/dev/null || true
rm -rf ~/.heliades

# Define chain parameters
CHAINID="42000"
MONIKER="helios-main-node"
# Initialize the chain with a moniker and chain ID
heliades init $MONIKER --chain-id $CHAINID

# Setup specifificities of the genesis and config
sh setup-helios-genesis-configuration.sh

# Define the keys array
KEYS=(
    "user0"
    "user1"
    "user2"
    "user3"
    "user4"
    "user5"
    "user6"
    "user7"
    "user8"
)


# Define mnemonics array (in same order as keys)
MNEMONICS=(
"web tail earth lesson domain feel slush bring amused repair lounge salt series stock fog remind ripple peace unknown sauce adjust blossom atom hotel"
"west mouse extra original dizzy dinosaur corn lottery access off slab surge piano build rabbit educate amused trophy orbit cable relax chimney trend inner"
"final rude almost banner language raven soon world aim pole copper poverty camera post wet humble hurt element find alone frown damp feature sadness"
"pupil target orient whip evidence life uniform mother senior strong lizard lens gesture young east armor loop library shadow fee host throw junk more"
"museum violin scan lonely knife fiscal ask science treat undo vacuum mention surge uniform mail tackle cricket artwork mimic alpha hero north before loan"
"lawsuit fame nice soft left method source ticket stage tourist unfold audit often reveal raise okay project absorb bubble spoon bounce track ready poet"
"slow fine dentist give small black shrug mouse fix coral omit type fish palace portion rhythm danger cream notice bless print pioneer announce course"
"liar oven damp useless again please dream birth box bottom hat olive slow rice busy atom carpet pilot always trust balcony hammer extend laptop"
"cash shoulder people eternal expire occur pen black funny idle afraid manual pause replace faith goose junior kite forest poet pulp treat cable merry"
)

# Import keys from mnemonics
for i in "${!KEYS[@]}"; do
    heliades keys add ${KEYS[$i]} --from-mnemonic "${MNEMONICS[$i]}" --keyring-backend="local"
done

# Integrate accounts into the genesis block with specifical balance
for key in "${KEYS[@]}"; do
    heliades add-genesis-account --chain-id $CHAINID $(heliades keys show $key -a --keyring-backend="local") 1000000000000000000000ahelios --keyring-backend="local"
done

echo "Signing genesis transaction"
# Register as Validator genesis account and delegate 900000 ahelios
heliades gentx user0 900000000000000000000ahelios --chain-id $CHAINID --keyring-backend="local" --gas-prices "1000000000ahelios"

echo "Collecting genesis transaction"
# Collect genesis Validators tx
heliades collect-gentxs

echo "Validating genesis"
# Validate the genesis file
heliades validate-genesis
echo "Setup done!"