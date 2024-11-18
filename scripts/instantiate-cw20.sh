CHAINID="4242"
PASSPHRASE="12345678"
USER=$(yes 12345678 | heliades keys show wasm -a)
INIT='{}'

vote() {
  ID=$1
  yes 12345678 | heliades tx gov vote $ID yes \
  --from genesis --keyring-backend file --gas=20000000 --fees=10000000000000000helios --yes \
  --chain-id 4242 --broadcast-mode sync
}


CW20_CODE_ID=1
echo "Store Wasm CW20 Code..."
yes $PASSPHRASE | heliades tx wasm store ./scripts/wasm-contracts/cw20_base.wasm --from=wasm --chain-id=$CHAINID --broadcast-mode=sync --gas=3000000 --fees=1500000000000000helios --yes
sleep 3

echo 'Instantiate a CW20 SOL token contract'
INIT='{"name":"CW20Solana","symbol":"SOL","decimals":6,"initial_balances":[{"address":"'$USER'","amount":"10000000000"}],"mint":{"minter":"'$USER'"},"marketing":{}}'
INSTANTIATE_TX_HASH=$(yes 12345678 | heliades tx wasm instantiate $CW20_CODE_ID "${INIT}" --label="CW20Solana" \
    --from=wasm --chain-id "${CHAINID}" --yes --no-admin \
    --fees=1000000000000000helios --gas=2000000 --from wasm | grep txhash | awk '{print $2}')
echo "INSTANTIATE_TX_HASH: $INSTANTIATE_TX_HASH"
sleep 3

ADAPTER_CODE_ID=2
echo "+ Store Cw20 adapter code"
yes $PASSPHRASE | heliades tx wasm store ./scripts/wasm-contracts/cw20_adapter.wasm --from=wasm --chain-id=$CHAINID --broadcast-mode=sync --fees=1500000000000000helios --gas=3000000 --yes
sleep 3

echo 'Instantiate CW20 Adapter contract'
INIT="{}"
INSTANTIATE_TX_HASH=$(yes $PASSPHRASE | heliades tx wasm instantiate $ADAPTER_CODE_ID "${INIT}" --label="CWAdapter" \
    --from=wasm --chain-id "${CHAINID}" --yes --from=wasm --no-admin \
    --fees=1000000000000000helios --gas=2000000 | grep txhash | awk '{print $2}')
sleep 3

echo 'Collect contract addresses...'
CW20_ADDRESS=$(heliades query wasm list-contract-by-code $CW20_CODE_ID --output json | jq -r '.contracts[-1]')
ADAPTER_ADDRESS=$(heliades query wasm list-contract-by-code $ADAPTER_CODE_ID --output json | jq -r '.contracts[-1]')

echo 'Fund adapter contract with helios'
yes $PASSPHRASE | heliades tx bank send wasm $ADAPTER_ADDRESS 100000000000000000000helios --from wasm --chain-id=$CHAINID --broadcast-mode=sync --gas=3000000 --fees=1500000000000000helios --yes
sleep 3

echo 'cw20 contract mint'
yes 12345678 | heliades tx wasm execute $CW20_ADDRESS \
'{"mint":{"recipient":"helios1cml96vmptgw99syqrrz8az79xer2pcgp0a885r","amount": "7777"}}' \
--from wasm --chain-id=4242 --fees=1500000000000000helios --yes --broadcast-mode=sync
sleep 3

echo 'cw20 contract transfer'
yes 12345678 | heliades tx wasm execute $CW20_ADDRESS \
'{"transfer":{"recipient":"helios1cml96vmptgw99syqrrz8az79xer2pcgp0a885r","amount": "9999"}}' \
--from wasm --chain-id=4242 --fees=1500000000000000helios --yes --broadcast-mode=sync
sleep 3

CODE_CREATOR=$(yes 12345678 | heliades keys show -a wasm)
PROPOSAL_ID=1
yes 12345678 | heliades tx wasm submit-proposal wasm-store ./scripts/wasm-contracts/cw20_base.wasm \
--title "Store CW20 base contract via proposal" \
--description "Store CW20 base contract" \
--deposit 500000000000000000000helios \
--from=wasm --chain-id="4242" --fees=10000000000000000helios \
--gas=20000000 \
--run-as $CODE_CREATOR \
--yes \
--broadcast-mode sync
vote $PROPOSAL_ID
