# CLI manual test for OCR

1. Start daemon

```sh
sh setup.sh


### add 5 keys for multisig ###
# add my keys
yes 12345678 | heliades keys add mykey
yes 12345678 | heliades keys add alicekey
yes 12345678 | heliades keys add bobkey
# add keys by other people
yes 12345678 | heliades keys add charliekey --pubkey='{"@type":"/helios.crypto.v1beta1.ethsecp256k1.PubKey","key":"AuJunSZYCMQBJHO1djMVYeDCG848kJZtUEpMPqOcpO5w"}'
yes 12345678 | heliades keys add carolkey --pubkey='{"@type":"/helios.crypto.v1beta1.ethsecp256k1.PubKey","key":"AhIEYBQqyWuXbIMR3iksHxaD5sFNhW1zyPUU3U5Qimzq"}'
### create multisig wallet (3/5) ###
yes 12345678 | heliades keys add feedadmin --multisig "mykey,alicekey,bobkey,charliekey,carolkey" --multisig-threshold 3
# address: helios1fadl0rutsh5a4pqn75g8wu62286w06yf9aaayy

# set feed admin address as environment variable
export FEEDADMIN=$(yes 12345678 | heliades keys show -a feedadmin)

# update module admin param
cat $HOME/.heliades/config/genesis.json | jq '.app_state["ocr"]["params"]["module_admin"]="'$FEEDADMIN'"' > $HOME/.heliades/config/tmp_genesis.json && mv $HOME/.heliades/config/tmp_genesis.json $HOME/.heliades/config/genesis.json

sh heliades.sh
```

2. On separate terminal

```sh
# set addresses as environment variable for future use
export SIGNER1=$(yes 12345678 | heliades keys show -a mykey)
export SIGNER2=$(yes 12345678 | heliades keys show -a alicekey)
export SIGNER3=$(yes 12345678 | heliades keys show -a bobkey)
export SIGNER4=$(yes 12345678 | heliades keys show -a charliekey)
export SIGNER5=$(yes 12345678 | heliades keys show -a carolkey)
export FEEDADMIN=$(yes 12345678 | heliades keys show -a feedadmin)

yes 12345678 | heliades tx bank send genesis $SIGNER1 1000000helios --chain-id=4242 --yes
yes 12345678 | heliades tx bank send genesis $SIGNER2 1000000helios --chain-id=4242 --yes
yes 12345678 | heliades tx bank send genesis $SIGNER3 1000000helios --chain-id=4242 --yes
yes 12345678 | heliades tx bank send genesis $SIGNER4 1000000helios --chain-id=4242 --yes
yes 12345678 | heliades tx bank send genesis $SIGNER5 1000000helios --chain-id=4242 --yes
yes 12345678 | heliades tx bank send genesis $FEEDADMIN 1000000helios --chain-id=4242 --yes


### get transaction file ###
heliades tx ocr create-feed --feed-id="BTC/USDT" --signers="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" --transmitters="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" --f=1 --offchain-config-version=1 --offchain-config="A641132V" --min-answer="0.01" --max-answer="100.0" --link-per-observation="10" --link-per-transmission="20" --link-denom="hyperion0x514910771AF9Ca656af840dff83E8264EcF986CA" --unique-reports=true --feed-config-description="BTC/USDT feed" --feed-admin=$FEEDADMIN  --billing-admin=$FEEDADMIN  --from=$FEEDADMIN  --yes --keyring-backend=file --chain-id=4242 --generate-only > createfeed.json

### get signatures of 3 keys (mykey, alicekey, bobkey) ###
yes 12345678 | heliades tx sign createfeed.json --from $SIGNER1 --chain-id=4242 --multisig=$FEEDADMIN > signature1.json
yes 12345678 | heliades tx sign createfeed.json --from $SIGNER2 --chain-id=4242 --multisig=$FEEDADMIN > signature2.json
yes 12345678 | heliades tx sign createfeed.json --from $SIGNER3 --chain-id=4242 --multisig=$FEEDADMIN > signature3.json

### merge signatures and make one transaction ###
yes 12345678 | heliades tx multisign createfeed.json feedadmin signature1.json signature2.json signature3.json --chain-id=4242 > signedcreatefeed.json

### broadcast the multisignature transaction ###
heliades tx broadcast signedcreatefeed.json

####### others for testing #######

### Try creating with non-module account ###
yes 12345678 | heliades tx ocr create-feed --feed-id="BTC/USDT" --signers="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" --transmitters="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" --f=1 --offchain-config-version=1 --offchain-config="A641132V" --min-answer="0.01" --max-answer="100.0" --link-per-observation="10" --link-per-transmission="20" --link-denom="hyperion0x514910771AF9Ca656af840dff83E8264EcF986CA" --unique-reports=true --feed-config-description="BTC/USDT feed" --feed-admin=$FEEDADMIN  --billing-admin=$FEEDADMIN  --from=genesis --keyring-backend=file --yes --chain-id=4242

### Create feed config with proposal ###
heliades tx ocr set-config-proposal \
    --title="set feed config" \
    --description="set feed config" \
    --deposit="100000000000helios" \
    --feed-id="BTC/USDT" \
    --signers="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
    --transmitters="$SIGNER1,$SIGNER2,$SIGNER3,$SIGNER4,$SIGNER5" \
    --f=1 --offchain-config-version=1 \
    --offchain-config="A641132V" \
    --min-answer="0.01" \
    --max-answer="100.0" \
    --link-per-observation="10" \
    --link-per-transmission="20" \
    --link-denom="hyperion0x514910771AF9Ca656af840dff83E8264EcF986CA" \
    --unique-reports=true \
    --feed-config-description="BTC/USDT feed" \
    --feed-admin=$FEEDADMIN \
    --billing-admin=$FEEDADMIN \
    --chain-id=4242 \
    --from=$FEEDADMIN \
    --yes
```
